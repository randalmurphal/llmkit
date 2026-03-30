package codexconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginAndMarketplaceRoundTrip(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins", "demo", ".codex-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("MkdirAll plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{"name":"demo","description":"Demo plugin","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	plugins, err := DiscoverPlugins(filepath.Join(root, "plugins"), PluginScopeRepo)
	if err != nil {
		t.Fatalf("DiscoverPlugins: %v", err)
	}
	if len(plugins) != 1 || plugins[0].Name != "demo" {
		t.Fatalf("unexpected plugins: %#v", plugins)
	}

	market := &Marketplace{
		Name: "local-repo",
		Plugins: []MarketplaceEntry{{
			Name: "demo",
			Source: MarketplaceSource{
				Source: "local",
				Path:   "./plugins/demo",
			},
		}},
	}
	path := filepath.Join(root, ".agents", "plugins", "marketplace.json")
	if err := SaveMarketplace(path, market); err != nil {
		t.Fatalf("SaveMarketplace: %v", err)
	}
	loaded, err := LoadMarketplace(path)
	if err != nil {
		t.Fatalf("LoadMarketplace: %v", err)
	}
	if FindMarketplaceEntry(loaded, "demo") == nil {
		t.Fatalf("expected marketplace entry for demo")
	}
}
