package codexconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Plugin struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version,omitempty"`
	Publisher   string      `json:"publisher,omitempty"`
	Path        string      `json:"-"`
	Scope       PluginScope `json:"-"`
}

type PluginScope string

const (
	PluginScopeRepo     PluginScope = "repo"
	PluginScopePersonal PluginScope = "personal"
)

type Marketplace struct {
	Name      string             `json:"name"`
	Interface *MarketplaceUI     `json:"interface,omitempty"`
	Plugins   []MarketplaceEntry `json:"plugins"`
}

type MarketplaceUI struct {
	DisplayName string `json:"displayName,omitempty"`
}

type MarketplaceEntry struct {
	Name     string            `json:"name"`
	Source   MarketplaceSource `json:"source"`
	Policy   map[string]string `json:"policy,omitempty"`
	Category string            `json:"category,omitempty"`
}

type MarketplaceSource struct {
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
}

func ParsePlugin(path string) (*Plugin, error) {
	filePath := path
	root := path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if info.IsDir() {
		if filepath.Base(path) == DirCodexPlugin {
			filePath = filepath.Join(path, FilePluginJSON)
			root = filepath.Dir(path)
		} else {
			filePath = filepath.Join(path, DirCodexPlugin, FilePluginJSON)
		}
	} else {
		root = filepath.Dir(filepath.Dir(path))
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read plugin.json: %w", err)
	}
	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, fmt.Errorf("parse plugin.json: %w", err)
	}
	plugin.Path = root
	return &plugin, nil
}

func DiscoverPlugins(root string, scope PluginScope) ([]*Plugin, error) {
	if !dirExists(root) {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}
	var plugins []*Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		plugin, err := ParsePlugin(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		plugin.Scope = scope
		plugins = append(plugins, plugin)
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })
	return plugins, nil
}

func LoadMarketplace(path string) (*Marketplace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Marketplace{}, nil
		}
		return nil, fmt.Errorf("read marketplace: %w", err)
	}
	var market Marketplace
	if err := json.Unmarshal(data, &market); err != nil {
		return nil, fmt.Errorf("parse marketplace: %w", err)
	}
	return &market, nil
}

func SaveMarketplace(path string, market *Marketplace) error {
	if market == nil {
		market = &Marketplace{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create marketplace dir: %w", err)
	}
	data, err := json.MarshalIndent(market, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal marketplace: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write marketplace: %w", err)
	}
	return nil
}

func FindMarketplaceEntry(market *Marketplace, name string) *MarketplaceEntry {
	if market == nil {
		return nil
	}
	for i := range market.Plugins {
		if strings.EqualFold(market.Plugins[i].Name, name) {
			return &market.Plugins[i]
		}
	}
	return nil
}
