package utils

import (
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func ResolveManifestPath(path string) (string, error) {
	if path == "" {
		path = "sai.sai"
	}
	return filepath.Abs(path)
}
