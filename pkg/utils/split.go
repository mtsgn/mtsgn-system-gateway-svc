package utils

import "strings"

func SplitPath(path string) []string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return []string{}
	}
	return strings.Split(cleanPath, "/")
}
