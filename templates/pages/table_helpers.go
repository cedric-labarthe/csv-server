package pages

import (
	"path"
	"unicode/utf8"
)

const maxCellChars = 100

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func parentURL(filePath string) string {
	dir := path.Dir(filePath)
	if dir == "." {
		return "/"
	}
	return "/browse/" + dir
}
