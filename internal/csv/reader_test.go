package csv

import (
	"testing"
)

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Basic alphabetic
		{"abc", "abd", true},
		{"abd", "abc", false},
		{"abc", "abc", false},

		// Case-insensitive ordering
		{"abc", "ABC", false}, // "abc" == "ABC" ignoring case; 'a'(97) > 'A'(65) so abc > ABC
		{"ABC", "abc", true},  // tie-breaker: 'A'(65) < 'a'(97)

		// The key bug cases: mixed case with differing digits
		{"A1", "a2", true},  // case-insensitive: A==a, then 1<2 → true
		{"A2", "a1", false}, // case-insensitive: A==a, then 2>1 → false
		{"a1", "A2", true},  // same as A1 vs a2
		{"a2", "A1", false}, // same as A2 vs a1

		// Tie-breaker: same content except case
		{"A1", "a1", true},  // case tie-breaker: 'A'(65) < 'a'(97)
		{"a1", "A1", false}, // case tie-breaker: 'a'(97) > 'A'(65)

		// Natural number ordering
		{"file2", "file10", true},
		{"file10", "file2", false},
		{"file10", "file10", false},

		// Numeric segments with leading zeros: the implementation uses digit-length
		// as a proxy for value, so "01" (len=2) sorts after "1" (len=1).
		{"file01", "file1", false}, // len("01")=2 > len("1")=1 → file01 > file1
		{"file1", "file01", true},  // len("1")=1 < len("01")=2 → file1 < file01

		// Empty strings
		{"", "", false},
		{"", "a", true},
		{"a", "", false},

		// Mixed digits and letters
		{"a1b", "a2b", true},
		{"a2b", "a1b", false},
		{"a1b", "a1b", false},

		// Case-insensitive with mixed digits at different positions
		{"File10", "file9", false}, // file==file (case), then 10>9 → false
		{"file9", "File10", true},  // file==file (case), then 9<10 → true
	}

	for _, tt := range tests {
		got := naturalLess(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
