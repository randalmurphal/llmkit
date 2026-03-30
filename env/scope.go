package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/randalmurphal/llmkit/v2/contract"
)

// ScopeConfig controls temporary provider-local mutations for a project.
type ScopeConfig struct {
	Hooks          map[string][]Hook                  `json:"hooks,omitempty"`
	MCPServers     map[string]contract.MCPServerConfig `json:"mcp_servers,omitempty"`
	Env            map[string]string                  `json:"env,omitempty"`
	Tag            string                             `json:"tag,omitempty"`
	RecoverOrphans bool                               `json:"recover_orphans,omitempty"`
	BackupSettings bool                               `json:"backup_settings,omitempty"`
}

// Scope tracks llmkit-owned project-local environment changes.
type Scope struct {
	provider string
	workDir  string
	record   scopeRecord

	mu       sync.Mutex
	restored bool
}

type scopeRegistry struct {
	Scopes map[string]scopeRecord `json:"scopes"`
}

type scopeRecord struct {
	Tag        string                            `json:"tag"`
	Provider   string                            `json:"provider"`
	WorkDir    string                            `json:"work_dir"`
	PID        int                               `json:"pid"`
	CreatedAt  time.Time                         `json:"created_at"`
	Hooks      map[string][]Hook                 `json:"hooks,omitempty"`
	MCPServers map[string]contract.MCPServerConfig `json:"mcp_servers,omitempty"`
	Env        map[string]string                 `json:"env,omitempty"`
}

// NewScope applies the requested project-local mutations and records them for later cleanup.
func NewScope(provider, workDir string, cfg ScopeConfig) (*Scope, error) {
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if workDir == "" {
		return nil, fmt.Errorf("workDir is required")
	}

	if cfg.RecoverOrphans {
		if err := recoverOrphanedScopes(provider, workDir); err != nil {
			return nil, err
		}
	}

	store, err := openProjectStore(provider, workDir)
	if err != nil {
		return nil, err
	}
	if cfg.BackupSettings {
		if err := backupPaths(store.paths()); err != nil {
			return nil, err
		}
	}

	record := scopeRecord{
		Tag:        cfg.Tag,
		Provider:   provider,
		WorkDir:    workDir,
		PID:        os.Getpid(),
		CreatedAt:  time.Now().UTC(),
		Hooks:      cloneHookMap(cfg.Hooks),
		MCPServers: cloneMCPServerMap(cfg.MCPServers),
		Env:        cloneStringMap(cfg.Env),
	}
	if record.Tag == "" {
		record.Tag = fmt.Sprintf("llmkit-%d-%d", record.PID, time.Now().UnixNano())
	}

	for event, hooks := range record.Hooks {
		for _, hook := range hooks {
			if err := store.addHook(event, hook); err != nil {
				return nil, err
			}
		}
	}
	for name, server := range record.MCPServers {
		if err := store.setMCP(name, server); err != nil {
			return nil, err
		}
	}
	for key, value := range record.Env {
		if err := store.setEnv(key, value); err != nil {
			return nil, err
		}
	}
	if err := store.save(); err != nil {
		return nil, err
	}

	reg, err := loadRegistry(workDir)
	if err != nil {
		return nil, err
	}
	reg.Scopes[record.Tag] = record
	if err := saveRegistry(workDir, reg); err != nil {
		return nil, err
	}

	return &Scope{
		provider: provider,
		workDir:  workDir,
		record:   record,
	}, nil
}

// Restore removes the exact llmkit-owned mutations tracked by the scope.
func (s *Scope) Restore() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if s.restored {
		s.mu.Unlock()
		return nil
	}
	s.restored = true
	s.mu.Unlock()

	if err := restoreRecord(s.workDir, s.record); err != nil {
		return err
	}

	reg, err := loadRegistry(s.workDir)
	if err == nil {
		delete(reg.Scopes, s.record.Tag)
		_ = saveRegistry(s.workDir, reg)
	}
	return nil
}

// Close is an io.Closer alias for Restore.
func (s *Scope) Close() error {
	return s.Restore()
}

func restoreRecord(workDir string, record scopeRecord) error {
	store, err := openProjectStore(record.Provider, workDir)
	if err != nil {
		return err
	}

	for event, hooks := range record.Hooks {
		for _, hook := range hooks {
			store.removeHookIfMatches(event, hook)
		}
	}
	for name, server := range record.MCPServers {
		store.removeMCPIfMatches(name, server)
	}
	for key, value := range record.Env {
		store.removeEnvIfMatches(key, value)
	}
	return store.save()
}

func recoverOrphanedScopes(provider, workDir string) error {
	reg, err := loadRegistry(workDir)
	if err != nil {
		return err
	}
	changed := false
	for tag, record := range reg.Scopes {
		if record.Provider != provider {
			continue
		}
		if processExists(record.PID) {
			continue
		}
		if err := restoreRecord(workDir, record); err != nil {
			return err
		}
		delete(reg.Scopes, tag)
		changed = true
	}
	if changed {
		return saveRegistry(workDir, reg)
	}
	return nil
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func loadRegistry(workDir string) (*scopeRegistry, error) {
	path := registryPath(workDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &scopeRegistry{Scopes: map[string]scopeRecord{}}, nil
		}
		return nil, err
	}
	var reg scopeRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg.Scopes == nil {
		reg.Scopes = map[string]scopeRecord{}
	}
	return &reg, nil
}

func saveRegistry(workDir string, reg *scopeRegistry) error {
	if reg == nil {
		reg = &scopeRegistry{Scopes: map[string]scopeRecord{}}
	}
	if reg.Scopes == nil {
		reg.Scopes = map[string]scopeRecord{}
	}
	return writeJSONAtomic(registryPath(workDir), reg)
}

func registryPath(workDir string) string {
	return filepath.Join(workDir, ".llmkit", "env-scopes.json")
}

func backupPaths(paths []string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func cloneHookMap(in map[string][]Hook) map[string][]Hook {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]Hook, len(in))
	for event, hooks := range in {
		out[event] = append([]Hook(nil), hooks...)
	}
	return out
}

func cloneMCPServerMap(in map[string]contract.MCPServerConfig) map[string]contract.MCPServerConfig {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]contract.MCPServerConfig, len(in))
	for name, server := range in {
		out[name] = cloneMCPServer(server)
	}
	return out
}
