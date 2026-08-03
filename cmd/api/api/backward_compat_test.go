package api

import (
	"encoding/json"
	"testing"

	"github.com/kernel/hypeman/lib/oapi"
	"github.com/stretchr/testify/require"
)

// TestHealthBackwardCompatible pins the health endpoint contract: it must
// keep returning exactly {"status":"ok"}.
func TestHealthBackwardCompatible(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)

	resp, err := svc.GetHealth(ctx(), oapi.GetHealthRequestObject{})
	require.NoError(t, err)
	okResp, ok := resp.(oapi.GetHealth200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)

	raw, err := json.Marshal(okResp)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"ok"}`, string(raw))
}

// TestResourcesBackwardCompatible pins the resources endpoint contract: the
// top-level fields existing clients consume must remain present.
func TestResourcesBackwardCompatible(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)

	// Wire the resource manager the same way production providers do.
	svc.ResourceManager.SetImageLister(svc.ImageManager)
	svc.ResourceManager.SetInstanceLister(svc.InstanceManager)
	svc.ResourceManager.SetVolumeLister(svc.VolumeManager)
	require.NoError(t, svc.ResourceManager.Initialize(ctx()))

	resp, err := svc.GetResources(ctx(), oapi.GetResourcesRequestObject{})
	require.NoError(t, err)
	okResp, ok := resp.(oapi.GetResources200JSONResponse)
	require.True(t, ok, "expected 200 response, got %T", resp)

	raw, err := json.Marshal(okResp)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, field := range []string{"cpu", "memory", "disk", "network", "allocations"} {
		require.Contains(t, decoded, field)
	}
}
