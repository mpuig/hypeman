package network

import (
	"context"
	"testing"

	"github.com/kernel/hypeman/cmd/api/config"
	"github.com/kernel/hypeman/lib/paths"
	"github.com/stretchr/testify/require"
)

// TestDefaultNetworkPrefersInitializedNetwork proves DefaultNetwork returns
// the effective default network established at Initialize time, including the
// guest-visible gateway, without touching host kernel state.
func TestDefaultNetworkPrefersInitializedNetwork(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	m := NewManager(paths.New(t.TempDir()), cfg, nil).(*manager)

	want := &Network{
		Name:     "default",
		Subnet:   "10.100.0.0/16",
		Gateway:  "10.100.0.1",
		Isolated: true,
		Default:  true,
	}
	m.setDefaultNetwork(want)

	nw, err := m.DefaultNetwork(context.Background())
	require.NoError(t, err)
	require.Equal(t, "10.100.0.1", nw.Gateway)
	require.Equal(t, "10.100.0.0/16", nw.Subnet)

	// Mutating the returned copy must not affect the cached network.
	nw.Gateway = " mutated "
	again, err := m.DefaultNetwork(context.Background())
	require.NoError(t, err)
	require.Equal(t, "10.100.0.1", again.Gateway)
}

func TestGuestToGuestEnabled(t *testing.T) {
	t.Parallel()
	require.False(t, GuestToGuestEnabled(&Network{Isolated: true}),
		"isolated default network blocks direct guest-to-guest traffic")
	if NetworkModel() == "bridge" {
		require.True(t, GuestToGuestEnabled(&Network{Isolated: false}))
	} else {
		require.False(t, GuestToGuestEnabled(&Network{Isolated: false}),
			"NAT networking never provides direct guest-to-guest reachability")
	}
}
