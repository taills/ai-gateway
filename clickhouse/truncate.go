package clickhouse

import "unicode/utf8"

// TruncateText returns the string capped at maxBytes UTF-8 bytes.
// If the string was truncated the second return value is true.
// The function ensures it does not split a multi-byte rune at the cut point.
func TruncateText(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s, false
	}
	// Walk back until we are on a valid rune boundary or reach index 0.
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	if maxBytes == 0 {
		return "", true
	}
	return s[:maxBytes], true
}
