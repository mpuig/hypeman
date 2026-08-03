package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuesRedactsCanary(t *testing.T) {
	const canary = "sk-canary-secret-value-12345"
	in := map[string]string{
		"API_KEY":     canary,
		"EMPTY_VALUE": "",
	}

	out := Values(in)

	require.Equal(t, map[string]string{
		"API_KEY":     Sentinel,
		"EMPTY_VALUE": Sentinel,
	}, out)
	for _, v := range out {
		require.NotContains(t, v, canary)
	}
	// Input map must not be mutated.
	require.Equal(t, canary, in["API_KEY"])
}

func TestValuesNil(t *testing.T) {
	require.Nil(t, Values(nil))
}

func TestIsSentinel(t *testing.T) {
	require.True(t, IsSentinel(Sentinel))
	require.False(t, IsSentinel(""))
	require.False(t, IsSentinel("real-value"))
}
