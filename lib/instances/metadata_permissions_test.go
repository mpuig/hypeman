package instances

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kernel/hypeman/cmd/api/config"
	"github.com/kernel/hypeman/lib/devices"
	"github.com/kernel/hypeman/lib/hypervisor"
	"github.com/kernel/hypeman/lib/images"
	"github.com/kernel/hypeman/lib/network"
	"github.com/kernel/hypeman/lib/paths"
	"github.com/kernel/hypeman/lib/redact"
	"github.com/kernel/hypeman/lib/system"
	"github.com/kernel/hypeman/lib/volumes"
	"github.com/stretchr/testify/require"
)

// newPermTestManager builds a minimal manager for metadata permission tests.
// It does not initialize networking or boot anything.
func newPermTestManager(t *testing.T, dataDir string) *manager {
	t.Helper()
	cfg := &config.Config{DataDir: dataDir}
	p := paths.New(dataDir)
	imageManager, err := images.NewManager(p, 1, nil)
	require.NoError(t, err)
	systemManager := system.NewManager(p)
	networkManager := network.NewManager(p, cfg, nil)
	deviceManager := devices.NewManager(p)
	volumeManager := volumes.NewManager(p, 0, nil)
	mgr := NewManager(p, imageManager, systemManager, networkManager, deviceManager, volumeManager,
		ResourceLimits{}, "", SnapshotPolicy{}, nil, nil).(*manager)
	return mgr
}

func metadataFixture(id string) *metadata {
	return &metadata{
		StoredMetadata: StoredMetadata{
			Id:             id,
			Name:           id,
			Image:          "docker.io/library/alpine:latest",
			Env:            map[string]string{"API_KEY": "secret-canary-value"},
			CreatedAt:      time.Now(),
			HypervisorType: hypervisor.TypeCloudHypervisor,
		},
	}
}

// TestSaveMetadataRestrictivePermissions proves instance metadata (which
// contains env values and credential bindings) is written owner-only.
func TestSaveMetadataRestrictivePermissions(t *testing.T) {
	t.Parallel()
	mgr := newPermTestManager(t, t.TempDir())
	meta := metadataFixture("inst-perms-save")

	require.NoError(t, mgr.ensureDirectories(meta.Id))
	require.NoError(t, mgr.saveMetadata(meta))

	info, err := os.Stat(mgr.paths.InstanceMetadata(meta.Id))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"instance metadata must be owner-only; it contains env values")
}

// TestManagerTightensLegacyMetadataPermissions proves the startup sweep
// upgrades metadata files written by older versions (mode 0644) to 0600.
func TestManagerTightensLegacyMetadataPermissions(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	p := paths.New(dataDir)

	// Simulate legacy metadata written with world-readable permissions.
	meta := metadataFixture("inst-perms-legacy")
	require.NoError(t, os.MkdirAll(p.InstanceDir(meta.Id), 0755))
	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p.InstanceMetadata(meta.Id), data, 0644))

	newPermTestManager(t, dataDir)

	info, err := os.Stat(p.InstanceMetadata(meta.Id))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"legacy 0644 metadata must be tightened to 0600 at startup")
}

// TestMergeEnvUpdateSkipsRedactionSentinel proves a redacted read response
// round-tripped into an env update cannot clobber real secret values.
func TestMergeEnvUpdateSkipsRedactionSentinel(t *testing.T) {
	t.Parallel()
	prev := map[string]string{"API_KEY": "real-secret", "PLAIN": "keep"}

	next := mergeEnvUpdate(prev, map[string]string{
		"API_KEY": redact.Sentinel, // redacted placeholder, must not overwrite
		"PLAIN":   "updated",
		"NEW":     "added",
	})

	require.Equal(t, map[string]string{
		"API_KEY": "real-secret",
		"PLAIN":   "updated",
		"NEW":     "added",
	}, next)
	// Previous map must not be mutated.
	require.Equal(t, "keep", prev["PLAIN"])

	// Nil previous env still yields a usable map.
	next = mergeEnvUpdate(nil, map[string]string{"A": "1"})
	require.Equal(t, map[string]string{"A": "1"}, next)
}

// TestWithoutRedactionSentinels proves an all-sentinel env patch collapses to
// no env update (so it skips the running-state and egress-proxy paths), while
// mixed patches keep only real values.
func TestWithoutRedactionSentinels(t *testing.T) {
	t.Parallel()

	require.Nil(t, withoutRedactionSentinels(map[string]string{
		"API_KEY": redact.Sentinel,
		"OTHER":   redact.Sentinel,
	}), "all-sentinel patch must not count as an env update")

	require.Equal(t, map[string]string{"PLAIN": "new"},
		withoutRedactionSentinels(map[string]string{
			"API_KEY": redact.Sentinel,
			"PLAIN":   "new",
		}))

	require.Nil(t, withoutRedactionSentinels(nil))
	require.Empty(t, withoutRedactionSentinels(map[string]string{}))
}
