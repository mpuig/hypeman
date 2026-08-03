// Package redact provides the shared sentinel and helpers for secret-safe
// diagnostics. Read APIs and log/diagnostic output use it to hide secret
// values (environment variables, credentials, tokens) while preserving keys
// so the output remains structurally useful.
package redact

// Sentinel replaces a secret value in redacted output. Keys are preserved so
// clients can see which variables exist without learning their values.
//
// The sentinel is treated as "no value provided" when it round-trips back
// into a mutation (for example an instance env update), so a redacted read
// response can never overwrite a real secret.
const Sentinel = "[redacted]"

// Values returns a copy of m with every value replaced by Sentinel.
// A nil input returns nil.
func Values(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k := range m {
		out[k] = Sentinel
	}
	return out
}

// IsSentinel reports whether v is the redaction sentinel placeholder.
func IsSentinel(v string) bool {
	return v == Sentinel
}
