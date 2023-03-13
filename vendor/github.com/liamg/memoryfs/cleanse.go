package memoryfs

import (
	"path/filepath"
	"strings"
)

func cleanse(path string) string {
	path = strings.ReplaceAll(path, "/", separator)
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, "."+separator)
	path = strings.TrimPrefix(path, separator)
	if path == "." {
		return ""
	}
	return path
}
