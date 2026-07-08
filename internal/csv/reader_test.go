package csv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

		// Numeric segments with leading zeros: "01" and "1" are numerically equal.
		{"file01", "file1", false},
		{"file1", "file01", false},

		// All-zero segments are numerically equal.
		{"file000", "file0", false},
		{"file0", "file000", false},

		// Non-ASCII digits (e.g. Arabic-Indic ١٢) are NOT treated as numeric
		// segments; they sort as plain characters.
		{"file١", "file٢", true},
		{"file2", "file١", true}, // ASCII '2'(50) < Arabic-Indic '١'(1633) — compared as chars

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

func TestSplitLeadingDigits(t *testing.T) {
	tests := []struct {
		input    string
		wantNum  string
		wantRest string
	}{
		{"123abc", "123", "abc"},
		{"abc", "", "abc"},
		{"123", "123", ""},
		{"", "", ""},
		{"0042rest", "0042", "rest"},
		{"1", "1", ""},
	}

	for _, tt := range tests {
		gotNum, gotRest := splitLeadingDigits(tt.input)
		if gotNum != tt.wantNum || gotRest != tt.wantRest {
			t.Errorf("splitLeadingDigits(%q) = (%q, %q), want (%q, %q)",
				tt.input, gotNum, gotRest, tt.wantNum, tt.wantRest)
		}
	}
}

func TestListEntriesNaturalSort(t *testing.T) {
	dir := t.TempDir()

	names := []string{
		"logfile-2026-10.csv",
		"logfile-2026-2.csv",
		"logfile-2026-6.csv",
		"logfile-2025-8.csv",
		"logfile-2025-9.csv",
		"logfile-2026-11.csv",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("a,b\n1,2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := ListEntries(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logfile-2025-8.csv",
		"logfile-2025-9.csv",
		"logfile-2026-2.csv",
		"logfile-2026-6.csv",
		"logfile-2026-10.csv",
		"logfile-2026-11.csv",
	}

	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, entry := range entries {
		if entry.Name != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, entry.Name, want[i])
		}
	}
}

func TestListEntriesDirsFirst(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "subdir2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "subdir10"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"file2.csv", "file10.csv"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("a\n1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := ListEntries(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		name  string
		isDir bool
	}{
		{"subdir2", true},
		{"subdir10", true},
		{"file2.csv", false},
		{"file10.csv", false},
	}

	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, entry := range entries {
		if entry.Name != want[i].name || entry.IsDir != want[i].isDir {
			t.Errorf("entries[%d] = {%q, isDir:%v}, want {%q, isDir:%v}",
				i, entry.Name, entry.IsDir, want[i].name, want[i].isDir)
		}
	}
}

func TestRepairRow(t *testing.T) {
	tests := []struct {
		name         string
		parts        []string
		expectedCols int
		delimiter    rune
		want         []string
	}{
		{
			name:         "exact match",
			parts:        []string{"a", "b", "c"},
			expectedCols: 3,
			delimiter:    ';',
			want:         []string{"a", "b", "c"},
		},
		{
			name:         "extra delimiter in second column",
			parts:        []string{"id", "foo", "bar", "baz", "tail1", "tail2", "tail3"},
			expectedCols: 5,
			delimiter:    ';',
			want:         []string{"id", "foo;bar;baz", "tail1", "tail2", "tail3"},
		},
		{
			name:         "one extra delimiter in second column",
			parts:        []string{"id", "foo", "bar", "c", "d", "e"},
			expectedCols: 5,
			delimiter:    ';',
			want:         []string{"id", "foo;bar", "c", "d", "e"},
		},
		{
			name:         "extra delimiter in last column fallback",
			parts:        []string{"only", "x", "y"},
			expectedCols: 2,
			delimiter:    ';',
			want:         []string{"only", "x;y"},
		},
		{
			name:         "too few parts padded",
			parts:        []string{"a", "b"},
			expectedCols: 4,
			delimiter:    ';',
			want:         []string{"a", "b", "", ""},
		},
		{
			name:         "single column with extras",
			parts:        []string{"a", "b", "c"},
			expectedCols: 1,
			delimiter:    ';',
			want:         []string{"a;b;c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repairRow(tt.parts, tt.expectedCols, tt.delimiter)
			if len(got) != len(tt.want) {
				t.Fatalf("repairRow() len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("repairRow()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReadTableRobustDelimiterInField(t *testing.T) {
	content := strings.Join([]string{
		"id;description;amount;status",
		"1;Alice;100;ok",
		"2;part1;part2;part3;100;active",
		"3;Carol;200;ok",
	}, "\n")

	fsys := fstest.MapFS{
		"test.csv": &fstest.MapFile{Data: []byte(content)},
	}

	table, err := ReadTable(fsys, "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	if len(table.Headers) != 4 {
		t.Fatalf("headers len = %d, want 4", len(table.Headers))
	}
	if len(table.Rows) != 3 {
		t.Fatalf("rows len = %d, want 3", len(table.Rows))
	}

	wantRow := []string{"2", "part1;part2;part3", "100", "active"}
	if len(table.Rows[1]) != len(wantRow) {
		t.Fatalf("row[1] len = %d, want %d (%v)", len(table.Rows[1]), len(wantRow), table.Rows[1])
	}
	for i, want := range wantRow {
		if table.Rows[1][i] != want {
			t.Errorf("row[1][%d] = %q, want %q", i, table.Rows[1][i], want)
		}
	}
}

func TestReadTableQuotedFieldsStillWork(t *testing.T) {
	content := "id;name;note\n1;Alice;\"hello;world\"\n"

	fsys := fstest.MapFS{
		"test.csv": &fstest.MapFile{Data: []byte(content)},
	}

	table, err := ReadTable(fsys, "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	if len(table.Rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(table.Rows))
	}
	if table.Rows[0][2] != "hello;world" {
		t.Errorf("quoted field = %q, want %q", table.Rows[0][2], "hello;world")
	}
}

func TestReadTableMultilineQuotedField(t *testing.T) {
	content := "id;name;note\n1;Alice;\"hello\nworld\"\n2;Bob;ok\n"

	fsys := fstest.MapFS{
		"test.csv": &fstest.MapFile{Data: []byte(content)},
	}

	table, err := ReadTable(fsys, "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	if len(table.Rows) != 2 {
		t.Fatalf("rows len = %d, want 2 (got %v)", len(table.Rows), table.Rows)
	}
	if table.Rows[0][2] != "hello\nworld" {
		t.Errorf("multiline quoted field = %q, want %q", table.Rows[0][2], "hello\nworld")
	}
}

func TestReadTableSkipsLeadingEmptyRows(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "blank delimiter row", data: ";\n\nid;name\n1;Alice\n"},
		{name: "multiple empty fields", data: ";;\n\nid;name\n1;Alice\n"},
		{name: "whitespace only line", data: "   \n\nid;name\n1;Alice\n"},
		{name: "bare newlines", data: "\n\nid;name\n1;Alice\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"test.csv": &fstest.MapFile{Data: []byte(tt.data)},
			}

			table, err := ReadTable(fsys, "test.csv")
			if err != nil {
				t.Fatal(err)
			}

			if len(table.Headers) != 2 || table.Headers[0] != "id" || table.Headers[1] != "name" {
				t.Fatalf("headers = %v, want [id name]", table.Headers)
			}
			if len(table.Rows) != 1 || table.Rows[0][0] != "1" || table.Rows[0][1] != "Alice" {
				t.Fatalf("rows = %v, want [[1 Alice]]", table.Rows)
			}
		})
	}
}
