package builds

import (
	"os"
	"testing"

	"github.com/kernel/hypeman/lib/paths"
	"github.com/stretchr/testify/require"
)

// TestWriteBuildConfigRestrictivePermissions proves build configs (which
// carry the registry push token) are owner-only, including when an existing
// world-readable file is rewritten in place (token refresh path).
func TestWriteBuildConfigRestrictivePermissions(t *testing.T) {
	t.Parallel()
	p := paths.New(t.TempDir())

	cfg := &BuildConfig{
		JobID:         "job-1",
		RegistryURL:   "localhost:8080",
		RegistryToken: "token-canary",
		SourcePath:    "/src",
		NetworkMode:   "isolated",
	}

	require.NoError(t, writeBuildConfig(p, "build-perms", cfg))
	info, err := os.Stat(p.BuildConfig("build-perms"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Simulate a legacy world-readable config, then rewrite it (token
	// refresh). WriteFile alone would keep 0644; the chmod must tighten it.
	require.NoError(t, os.Chmod(p.BuildConfig("build-perms"), 0644))
	require.NoError(t, writeBuildConfig(p, "build-perms", cfg))
	info, err = os.Stat(p.BuildConfig("build-perms"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"rewritten build config must be tightened to 0600")
}

// TestManagerTightensLegacyBuildFilePermissions proves the startup sweep
// upgrades build config/metadata files written by older versions to 0600.
func TestManagerTightensLegacyBuildFilePermissions(t *testing.T) {
	t.Parallel()
	p := paths.New(t.TempDir())

	require.NoError(t, os.MkdirAll(p.BuildDir("build-legacy"), 0755))
	require.NoError(t, os.WriteFile(p.BuildConfig("build-legacy"), []byte(`{"job_id":"j"}`), 0644))
	require.NoError(t, os.WriteFile(p.BuildMetadata("build-legacy"), []byte(`{"id":"build-legacy"}`), 0644))

	m := &manager{paths: p}
	m.tightenBuildFilePermissions()

	for _, path := range []string{p.BuildConfig("build-legacy"), p.BuildMetadata("build-legacy")} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm(), "%s must be tightened to 0600", path)
	}
}
