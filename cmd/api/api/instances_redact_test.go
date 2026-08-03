package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kernel/hypeman/lib/hypervisor"
	"github.com/kernel/hypeman/lib/instances"
	mw "github.com/kernel/hypeman/lib/middleware"
	"github.com/kernel/hypeman/lib/oapi"
	"github.com/kernel/hypeman/lib/paths"
	"github.com/kernel/hypeman/lib/redact"
	"github.com/stretchr/testify/require"
)

const envCanaryValue = "sk-canary-env-value-do-not-leak-7f3a9b"

// writeInstanceMetadataFixture writes an instance metadata.json directly into
// the data dir so read APIs can be tested without booting a VM.
func writeInstanceMetadataFixture(t *testing.T, dataDir string, meta instances.StoredMetadata) {
	t.Helper()
	p := paths.New(dataDir)
	require.NoError(t, os.MkdirAll(p.InstanceDir(meta.Id), 0755))
	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p.InstanceMetadata(meta.Id), data, 0600))
}

func envCanaryInstance() instances.Instance {
	return instances.Instance{
		StoredMetadata: instances.StoredMetadata{
			Id:             "inst-env-canary",
			Name:           "env-canary",
			Image:          "docker.io/library/alpine:latest",
			Env:            map[string]string{"API_KEY": envCanaryValue, "PLAIN": "hello"},
			CreatedAt:      time.Now(),
			HypervisorType: hypervisor.TypeCloudHypervisor,
		},
		State: instances.StateStopped,
	}
}

// TestListInstancesRedactsEnvByDefault proves the raw API no longer exposes
// env values on reads: keys are preserved, values are redacted, and the
// canary never appears unless include_env=true is passed.
func TestListInstancesRedactsEnvByDefault(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	inst := envCanaryInstance()
	writeInstanceMetadataFixture(t, svc.Config.DataDir, inst.StoredMetadata)

	// Default: redacted.
	resp, err := svc.ListInstances(ctx(), oapi.ListInstancesRequestObject{})
	require.NoError(t, err)
	list, ok := resp.(oapi.ListInstances200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].Env)
	require.Equal(t, redact.Sentinel, (*list[0].Env)["API_KEY"])
	require.Equal(t, redact.Sentinel, (*list[0].Env)["PLAIN"])
	raw, err := json.Marshal(list)
	require.NoError(t, err)
	require.NotContains(t, string(raw), envCanaryValue, "redacted list response leaked env canary")

	// Opt-in: plaintext.
	include := true
	resp, err = svc.ListInstances(ctx(), oapi.ListInstancesRequestObject{
		Params: oapi.ListInstancesParams{IncludeEnv: &include},
	})
	require.NoError(t, err)
	list, ok = resp.(oapi.ListInstances200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Len(t, list, 1)
	require.Equal(t, envCanaryValue, (*list[0].Env)["API_KEY"])
}

// TestGetInstanceRedactsEnvByDefault covers the single-instance read path.
func TestGetInstanceRedactsEnvByDefault(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	inst := envCanaryInstance()

	resolvedCtx := mw.WithResolvedInstance(ctx(), inst.Id, inst)

	resp, err := svc.GetInstance(resolvedCtx, oapi.GetInstanceRequestObject{Id: inst.Id})
	require.NoError(t, err)
	got, ok := resp.(oapi.GetInstance200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.NotNil(t, got.Env)
	require.Equal(t, redact.Sentinel, (*got.Env)["API_KEY"])
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), envCanaryValue, "redacted get response leaked env canary")

	include := true
	resp, err = svc.GetInstance(resolvedCtx, oapi.GetInstanceRequestObject{
		Id:     inst.Id,
		Params: oapi.GetInstanceParams{IncludeEnv: &include},
	})
	require.NoError(t, err)
	got, ok = resp.(oapi.GetInstance200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)
	require.Equal(t, envCanaryValue, (*got.Env)["API_KEY"])
}

func TestIncludeEnvParam(t *testing.T) {
	t.Parallel()
	require.False(t, includeEnv(nil))
	f := false
	require.False(t, includeEnv(&f))
	tr := true
	require.True(t, includeEnv(&tr))
}

// TestMetadataFixturePerms keeps the fixture helper honest: fixtures on disk
// must carry the sentinel value so the redaction assertions above are
// meaningful.
func TestMetadataFixturePerms(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeInstanceMetadataFixture(t, dir, envCanaryInstance().StoredMetadata)
	data, err := os.ReadFile(filepath.Join(dir, "guests", "inst-env-canary", "metadata.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), envCanaryValue)
}
