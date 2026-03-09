package components

import "strings"

func BreadcrumbTitle(currentPath string) string {
	if currentPath == "" {
		return "CSV Viewer"
	}
	parts := strings.Split(currentPath, "/")
	return parts[len(parts)-1]
}

func JoinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}
