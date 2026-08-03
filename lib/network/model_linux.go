//go:build linux

package network

// NetworkModel identifies the guest networking model in use on this host.
// Linux hosts use a bridge with per-VM TAP devices and port isolation.
func NetworkModel() string {
	return "bridge"
}

// GuestToGuestEnabled reports whether direct VM-to-VM traffic is permitted on
// the given default network. Linux default networks use per-TAP port
// isolation, so guests cannot reach each other directly.
func GuestToGuestEnabled(n *Network) bool {
	return n != nil && !n.Isolated
}

// preferCachedDefaultNetwork reports whether DefaultNetwork should trust the
// network cached during Initialize. On Linux the cache mirrors the bridge
// built from config, so it is authoritative.
func preferCachedDefaultNetwork() bool {
	return true
}
