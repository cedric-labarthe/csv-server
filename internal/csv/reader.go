package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

type Entry struct {
	Name  string
	IsDir bool
	Size  int64 // bytes; set for files only
}

// ListEntries returns all directories and .csv files in dir, directories first.
func ListEntries(dir string) ([]Entry, error) {
	osEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	entries := make([]Entry, 0, len(osEntries))
	for _, osEntry := range osEntries {
		if osEntry.IsDir() {
			entries = append(entries, Entry{Name: osEntry.Name(), IsDir: true})
		} else if strings.EqualFold(filepath.Ext(osEntry.Name()), ".csv") {
			info, err := osEntry.Info()
			if err != nil {
				return nil, fmt.Errorf("reading file info for %q: %w", osEntry.Name(), err)
			}
			entries = append(entries, Entry{Name: osEntry.Name(), IsDir: false, Size: info.Size()})
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		nameI, nameJ := entries[i].Name, entries[j].Name
		if naturalLess(nameI, nameJ) {
			return true
		}
		if naturalLess(nameJ, nameI) {
			return false
		}
		return nameI < nameJ
	})

	return entries, nil
}

// naturalLess compares two strings using natural sort order so that numeric
// segments are compared by value ("2" < "10") instead of lexicographically.
// Alphabetic comparisons are case-insensitive; when two strings differ only in
// case, the one whose first differing rune has the lower Unicode code point
// sorts first (deterministic tie-breaker).
func naturalLess(a, b string) bool {
	// firstCaseDiff records the ordering of the first position where the runes
	// differ only in case: -1 means a's rune < b's rune, +1 means a's rune > b's rune.
	firstCaseDiff := 0
	for {
		if b == "" {
			if a == "" {
				// Strings are equal ignoring case; use case tie-breaker.
				return firstCaseDiff < 0
			}
			return false
		}
		if a == "" {
			return true
		}

		aRune, aWidth := utf8.DecodeRuneInString(a)
		bRune, bWidth := utf8.DecodeRuneInString(b)

		aIsDigit := aRune >= '0' && aRune <= '9'
		bIsDigit := bRune >= '0' && bRune <= '9'

		if aIsDigit && bIsDigit {
			aNum, aRest := splitLeadingDigits(a)
			bNum, bRest := splitLeadingDigits(b)
			aTrim := strings.TrimLeft(aNum, "0")
			bTrim := strings.TrimLeft(bNum, "0")
			if aTrim == "" {
				aTrim = "0"
			}
			if bTrim == "" {
				bTrim = "0"
			}
			if len(aTrim) != len(bTrim) {
				return len(aTrim) < len(bTrim)
			}
			if aTrim != bTrim {
				return aTrim < bTrim
			}
			a, b = aRest, bRest
			continue
		}

		aLow, bLow := unicode.ToLower(aRune), unicode.ToLower(bRune)
		if aLow != bLow {
			return aLow < bLow
		}
		// Same case-folded rune: record the first case-only difference as a
		// deterministic tie-breaker and continue scanning.
		if aRune != bRune && firstCaseDiff == 0 {
			if aRune < bRune {
				firstCaseDiff = -1
			} else {
				firstCaseDiff = 1
			}
		}
		a, b = a[aWidth:], b[bWidth:]
	}
}

func splitLeadingDigits(s string) (digits, rest string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:]
}

// ReadTable parses a CSV file from fsys and returns its headers and rows.
func ReadTable(fsys fs.FS, name string) (Table, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return Table{}, fmt.Errorf("opening %q: %w", name, err)
	}
	defer f.Close()

	return parseTable(f)
}

func parseTable(r io.Reader) (Table, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Table{}, fmt.Errorf("reading content: %w", err)
	}

	delimiter := detectDelimiter(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	headers, err := readFirstNonEmptyRow(reader)
	if err != nil {
		if err == io.EOF {
			return Table{}, fmt.Errorf("reading headers: empty file")
		}
		return Table{}, fmt.Errorf("reading headers: %w", err)
	}
	expectedCols := len(headers)

	rows := make([][]string, 0)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Table{}, fmt.Errorf("reading rows: %w", err)
		}
		if len(row) != expectedCols {
			row = repairRow(row, expectedCols, delimiter)
		}
		if !isEmptyRow(row) {
			rows = append(rows, row)
		}
	}

	return Table{Headers: headers, Rows: rows}, nil
}

func readFirstNonEmptyRow(reader *csv.Reader) ([]string, error) {
	for {
		row, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if !isEmptyRow(row) {
			return row, nil
		}
	}
}

// repairRow fixes rows split on too many delimiters. Surplus parts are merged
// back into a single field. When expectedCols >= 2, extras are assumed to sit
// in the second column (common for description fields). Otherwise the surplus
// is merged into the last column. Rows with too few parts are padded.
func repairRow(parts []string, expectedCols int, delimiter rune) []string {
	if len(parts) == expectedCols {
		return parts
	}
	if len(parts) < expectedCols {
		out := make([]string, expectedCols)
		copy(out, parts)
		return out
	}

	delim := string(delimiter)
	extra := len(parts) - expectedCols

	if expectedCols == 1 {
		return []string{strings.Join(parts, delim)}
	}

	// Merge surplus delimiters into the second column (index 1).
	mergeEnd := 1 + extra + 1
	out := make([]string, 0, expectedCols)
	out = append(out, parts[0])
	out = append(out, strings.Join(parts[1:mergeEnd], delim))
	out = append(out, parts[mergeEnd:]...)
	if len(out) == expectedCols {
		return out
	}

	// Fallback: merge surplus into the last column.
	merged := strings.Join(parts[expectedCols-1:], delim)
	return append(parts[:expectedCols-1], merged)
}

// detectDelimiter scans the first non-empty line and picks the most frequent
// candidate among ';', ',', '\t', '|'.
func detectDelimiter(data []byte) rune {
	candidates := []rune{';', ',', '\t', '|'}

	for line := range nonEmptyLines(data) {
		best, bestCount := ',', 0
		for _, c := range candidates {
			count := strings.Count(line, string(c))
			if count > bestCount {
				best, bestCount = c, count
			}
		}
		return best
	}

	return ','
}

func nonEmptyLines(data []byte) iter.Seq[string] {
	return func(yield func(string) bool) {
		for len(data) > 0 {
			idx := bytes.IndexByte(data, '\n')
			var line []byte
			if idx < 0 {
				line = data
				data = nil
			} else {
				line = data[:idx]
				data = data[idx+1:]
			}
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			trimmed := strings.TrimSpace(string(line))
			if trimmed != "" && !yield(trimmed) {
				return
			}
			if idx < 0 {
				return
			}
		}
	}
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
