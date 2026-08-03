package builds

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kernel/hypeman/lib/paths"
	"github.com/stretchr/testify/require"
)

func TestBuildMetadataReadWrite_MetadataRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	p := paths.New(tempDir)
	id := "build-meta-1"

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "builds", id), 0755))

	meta := &buildMetadata{
		ID:        id,
		Status:    StatusQueued,
		Tags:      map[string]string{"team": "backend", "env": "staging"},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	require.NoError(t, writeMetadata(p, meta))

	loaded, err := readMetadata(p, id)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"team": "backend", "env": "staging"}, loaded.Tags)

	build := loaded.toBuild()
	require.Equal(t, map[string]string{"team": "backend", "env": "staging"}, build.Tags)

	loaded.Tags["team"] = "mutated"
	require.Equal(t, "backend", build.Tags["team"])
}

func TestWriteBuildConfig_UsesOwnerOnlyPermissions(t *testing.T) {
	tempDir := t.TempDir()
	p := paths.New(tempDir)
	id := "build-config-1"

	cfg := &BuildConfig{RegistryToken: "secret-token"}
	require.NoError(t, writeBuildConfig(p, id, cfg))

	info, err := os.Stat(p.BuildConfig(id))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestWriteBuildConfig_TightensLegacyPermissions(t *testing.T) {
	tempDir := t.TempDir()
	p := paths.New(tempDir)
	id := "build-config-legacy"

	require.NoError(t, os.MkdirAll(p.BuildDir(id), 0755))
	configPath := p.BuildConfig(id)
	require.NoError(t, os.WriteFile(configPath, []byte(`{"registry_token":"old-token"}`), 0644))
	require.NoError(t, os.Chmod(configPath, 0644))

	cfg := &BuildConfig{RegistryToken: "new-token"}
	require.NoError(t, writeBuildConfig(p, id, cfg))

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
