package csv

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
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
			entries = append(entries, Entry{Name: osEntry.Name(), IsDir: false})
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

		aIsDigit := unicode.IsDigit(aRune)
		bIsDigit := unicode.IsDigit(bRune)

		if aIsDigit && bIsDigit {
			aNum, aRest := splitLeadingDigits(a)
			bNum, bRest := splitLeadingDigits(b)
			aTrim := strings.TrimLeft(aNum, "0")
			bTrim := strings.TrimLeft(bNum, "0")
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
	for i < len(s) {
		r, width := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsDigit(r) {
			break
		}
		i += width
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
	// Buffer the content to allow peeking for delimiter detection.
	raw, err := io.ReadAll(r)
	if err != nil {
		return Table{}, fmt.Errorf("reading content: %w", err)
	}

	delimiter := detectDelimiter(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := readFirstNonEmptyRow(reader)
	if err != nil {
		return Table{}, fmt.Errorf("reading headers: %w", err)
	}

	allRows, err := reader.ReadAll()
	if err != nil {
		return Table{}, fmt.Errorf("reading rows: %w", err)
	}

	rows := make([][]string, 0, len(allRows))
	for _, row := range allRows {
		if !isEmptyRow(row) {
			rows = append(rows, row)
		}
	}

	return Table{Headers: headers, Rows: rows}, nil
}

// detectDelimiter scans the first non-empty line and picks the most frequent
// candidate among ';', ',', '\t', '|'.
func detectDelimiter(data []byte) rune {
	candidates := []rune{';', ',', '\t', '|'}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
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

// readFirstNonEmptyRow skips blank rows and returns the first row with data.
func readFirstNonEmptyRow(r *csv.Reader) ([]string, error) {
	for {
		row, err := r.Read()
		if err != nil {
			return nil, err
		}
		if !isEmptyRow(row) {
			return row, nil
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
