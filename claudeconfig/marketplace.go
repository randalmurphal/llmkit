package claudeconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultMarketplaceURL is the default Claude Code plugin marketplace.
const DefaultMarketplaceURL = "https://plugins.claude.ai/api/v1"

// DefaultCacheTTL is the default cache duration for marketplace data.
const DefaultCacheTTL = 15 * time.Minute

// MarketplacePlugin represents a plugin available in the marketplace.
type MarketplacePlugin struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Author      PluginAuthor `json:"author"`
	Version     string       `json:"version"`
	Repository  string       `json:"repository,omitempty"`
	Downloads   int          `json:"downloads,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Homepage    string       `json:"homepage,omitempty"`
	UpdatedAt   time.Time    `json:"updated_at,omitempty"`
}

// PluginUpdateInfo contains information about an available update.
type PluginUpdateInfo struct {
	Name           string      `json:"name"`
	CurrentVersion string      `json:"current_version"`
	LatestVersion  string      `json:"latest_version"`
	Scope          PluginScope `json:"scope"`
}

// MarketplaceCache stores cached marketplace data.
type MarketplaceCache struct {
	Plugins   []MarketplacePlugin `json:"plugins"`
	UpdatedAt time.Time           `json:"updated_at"`
	TTL       time.Duration       `json:"-"`
}

// IsValid returns true if the cache is still valid.
func (c *MarketplaceCache) IsValid() bool {
	if c == nil || len(c.Plugins) == 0 {
		return false
	}
	return time.Since(c.UpdatedAt) < c.TTL
}

// MarketplaceService handles marketplace operations.
type MarketplaceService struct {
	claudeDir    string
	marketplaceURL string
	cache        *MarketplaceCache
	cachePath    string
	httpClient   *http.Client
	mu           sync.RWMutex
}

// MarketplaceOption configures the MarketplaceService.
type MarketplaceOption func(*MarketplaceService)

// WithMarketplaceURL sets a custom marketplace URL.
func WithMarketplaceURL(url string) MarketplaceOption {
	return func(s *MarketplaceService) {
		s.marketplaceURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) MarketplaceOption {
	return func(s *MarketplaceService) {
		s.httpClient = client
	}
}

// WithCacheTTL sets the cache TTL.
func WithCacheTTL(ttl time.Duration) MarketplaceOption {
	return func(s *MarketplaceService) {
		if s.cache != nil {
			s.cache.TTL = ttl
		}
	}
}

// NewMarketplaceService creates a new MarketplaceService.
func NewMarketplaceService(claudeDir string, opts ...MarketplaceOption) *MarketplaceService {
	s := &MarketplaceService{
		claudeDir:      claudeDir,
		marketplaceURL: DefaultMarketplaceURL,
		cachePath:      filepath.Join(claudeDir, "plugins", "marketplace_cache.json"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &MarketplaceCache{
			TTL: DefaultCacheTTL,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Try to load existing cache
	_ = s.loadCache()

	return s
}

// Browse returns available plugins from the marketplace.
func (s *MarketplaceService) Browse(page, limit int) ([]MarketplacePlugin, int, error) {
	s.mu.RLock()
	cacheValid := s.cache.IsValid()
	s.mu.RUnlock()

	if !cacheValid {
		if err := s.RefreshCache(); err != nil {
			// Return cached data if available, even if stale
			s.mu.RLock()
			defer s.mu.RUnlock()
			if len(s.cache.Plugins) > 0 {
				return s.paginatePlugins(s.cache.Plugins, page, limit), len(s.cache.Plugins), nil
			}
			return nil, 0, fmt.Errorf("fetch marketplace: %w", err)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.paginatePlugins(s.cache.Plugins, page, limit), len(s.cache.Plugins), nil
}

// paginatePlugins returns a slice of plugins for the given page.
func (s *MarketplaceService) paginatePlugins(plugins []MarketplacePlugin, page, limit int) []MarketplacePlugin {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * limit
	if start >= len(plugins) {
		return []MarketplacePlugin{}
	}

	end := start + limit
	if end > len(plugins) {
		end = len(plugins)
	}

	return plugins[start:end]
}

// Search searches for plugins matching the query.
func (s *MarketplaceService) Search(query string) ([]MarketplacePlugin, error) {
	// Ensure cache is populated
	s.mu.RLock()
	cacheValid := s.cache.IsValid()
	s.mu.RUnlock()

	if !cacheValid {
		if err := s.RefreshCache(); err != nil {
			s.mu.RLock()
			defer s.mu.RUnlock()
			if len(s.cache.Plugins) == 0 {
				return nil, fmt.Errorf("fetch marketplace: %w", err)
			}
			// Continue with stale cache
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Simple substring search on name, description, and keywords
	var results []MarketplacePlugin
	query = url.QueryEscape(query) // For API calls
	queryLower := query

	for _, p := range s.cache.Plugins {
		if containsIgnoreCase(p.Name, queryLower) ||
			containsIgnoreCase(p.Description, queryLower) ||
			containsAnyIgnoreCase(p.Keywords, queryLower) {
			results = append(results, p)
		}
	}

	return results, nil
}

// GetPlugin fetches details for a specific plugin from the marketplace.
func (s *MarketplaceService) GetPlugin(name string) (*MarketplacePlugin, error) {
	// First check cache
	s.mu.RLock()
	for _, p := range s.cache.Plugins {
		if p.Name == name {
			s.mu.RUnlock()
			return &p, nil
		}
	}
	s.mu.RUnlock()

	// Fetch from API
	resp, err := s.httpClient.Get(s.marketplaceURL + "/plugins/" + url.PathEscape(name))
	if err != nil {
		return nil, fmt.Errorf("fetch plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var plugin MarketplacePlugin
	if err := json.NewDecoder(resp.Body).Decode(&plugin); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &plugin, nil
}

// Install downloads and installs a plugin from the marketplace.
func (s *MarketplaceService) Install(name, version string, scope PluginScope, projectRoot string) (*Plugin, error) {
	// Determine destination directory
	var destDir string
	switch scope {
	case PluginScopeGlobal:
		globalDir, err := GlobalPluginsDir()
		if err != nil {
			return nil, fmt.Errorf("get global plugins dir: %w", err)
		}
		destDir = filepath.Join(globalDir, name)
	case PluginScopeProject:
		destDir = filepath.Join(ProjectPluginsDir(projectRoot), name)
	default:
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	// Check if already installed
	if dirExists(destDir) {
		return nil, fmt.Errorf("plugin already installed: %s", name)
	}

	// Download plugin
	downloadURL := fmt.Sprintf("%s/plugins/%s/download", s.marketplaceURL, url.PathEscape(name))
	if version != "" {
		downloadURL += "?version=" + url.QueryEscape(version)
	}

	resp, err := s.httpClient.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create plugin directory: %w", err)
	}

	// Extract tarball (assuming tar.gz format)
	if err := extractTarGz(resp.Body, destDir); err != nil {
		// Clean up on failure
		_ = os.RemoveAll(destDir)
		return nil, fmt.Errorf("extract plugin: %w", err)
	}

	// Parse the installed plugin
	plugin, err := ParsePluginJSON(destDir)
	if err != nil {
		_ = os.RemoveAll(destDir)
		return nil, fmt.Errorf("parse installed plugin: %w", err)
	}

	plugin.Scope = scope
	plugin.InstalledAt = time.Now()

	return plugin, nil
}

// CheckUpdates checks for available updates for installed plugins.
func (s *MarketplaceService) CheckUpdates(plugins []*Plugin) ([]PluginUpdateInfo, error) {
	// Ensure cache is fresh
	if err := s.RefreshCache(); err != nil {
		// Continue with stale cache if available
		s.mu.RLock()
		if len(s.cache.Plugins) == 0 {
			s.mu.RUnlock()
			return nil, fmt.Errorf("fetch marketplace: %w", err)
		}
		s.mu.RUnlock()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build lookup map
	marketplaceVersions := make(map[string]string)
	for _, p := range s.cache.Plugins {
		marketplaceVersions[p.Name] = p.Version
	}

	var updates []PluginUpdateInfo
	for _, p := range plugins {
		if latestVersion, ok := marketplaceVersions[p.Name]; ok {
			if p.Version != "" && latestVersion != "" && p.Version != latestVersion {
				updates = append(updates, PluginUpdateInfo{
					Name:           p.Name,
					CurrentVersion: p.Version,
					LatestVersion:  latestVersion,
					Scope:          p.Scope,
				})
			}
		}
	}

	return updates, nil
}

// RefreshCache forces a cache refresh from the marketplace.
func (s *MarketplaceService) RefreshCache() error {
	resp, err := s.httpClient.Get(s.marketplaceURL + "/plugins")
	if err != nil {
		return fmt.Errorf("fetch plugins list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var plugins []MarketplacePlugin
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	s.mu.Lock()
	s.cache.Plugins = plugins
	s.cache.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Persist cache
	_ = s.saveCache()

	return nil
}

// loadCache loads the cache from disk.
func (s *MarketplaceService) loadCache() error {
	data, err := os.ReadFile(s.cachePath)
	if err != nil {
		return err
	}

	var cache MarketplaceCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}

	s.mu.Lock()
	cache.TTL = s.cache.TTL // Preserve configured TTL
	s.cache = &cache
	s.mu.Unlock()

	return nil
}

// saveCache persists the cache to disk.
func (s *MarketplaceService) saveCache() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.cache, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.cachePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(s.cachePath, data, 0644)
}

// CacheAge returns how old the cache is.
func (s *MarketplaceService) CacheAge() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.cache.UpdatedAt)
}

// IsCacheValid returns whether the cache is still valid.
func (s *MarketplaceService) IsCacheValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache.IsValid()
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		(len(s) > 0 && containsLower(toLower(s), toLower(substr))))
}

func containsLower(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsAnyIgnoreCase(slice []string, substr string) bool {
	for _, s := range slice {
		if containsIgnoreCase(s, substr) {
			return true
		}
	}
	return false
}

// extractTarGz extracts a tar.gz archive to the destination directory.
// This is a placeholder - actual implementation would use archive/tar and compress/gzip.
func extractTarGz(r io.Reader, destDir string) error {
	// For now, return an error indicating this needs implementation
	// In a real implementation, this would use:
	// - compress/gzip to decompress
	// - archive/tar to extract files
	return fmt.Errorf("tar.gz extraction not implemented - use git clone or manual install")
}
