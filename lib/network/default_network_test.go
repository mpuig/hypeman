package network

import (
	"context"
	"testing"

	"github.com/kernel/hypeman/cmd/api/config"
	"github.com/kernel/hypeman/lib/paths"
	"github.com/stretchr/testify/require"
)

// TestDefaultNetworkEffectiveGateway proves DefaultNetwork reports the
// guest-visible gateway for the host's networking model: the initialized
// (config-derived) network on Linux, and the vz NAT stub on macOS, where the
// configured subnet/gateway are ignored.
func TestDefaultNetworkEffectiveGateway(t *testing.T) {
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
	if preferCachedDefaultNetwork() {
		require.Equal(t, "10.100.0.1", nw.Gateway)
		require.Equal(t, "10.100.0.0/16", nw.Subnet)

		// Mutating the returned copy must not affect the cached network.
		nw.Gateway = " mutated "
		again, err := m.DefaultNetwork(context.Background())
		require.NoError(t, err)
		require.Equal(t, "10.100.0.1", again.Gateway)
	} else {
		// macOS: the vz NAT stub is authoritative; the config-derived cache
		// is never guest-visible.
		require.Equal(t, "192.168.64.1", nw.Gateway)
		require.Equal(t, "192.168.64.0/24", nw.Subnet)
	}
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
