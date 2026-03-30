package env

import (
	"os"
	"sync"
)

var (
	tempFilesMu sync.Mutex
	tempFiles   = map[string]struct{}{}
)

// TempFile creates a tracked temp file.
func TempFile(dir, pattern string) (*os.File, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	tempFilesMu.Lock()
	tempFiles[file.Name()] = struct{}{}
	tempFilesMu.Unlock()
	return file, nil
}

// Cleanup removes all tracked temp files.
func Cleanup() {
	tempFilesMu.Lock()
	paths := make([]string, 0, len(tempFiles))
	for path := range tempFiles {
		paths = append(paths, path)
	}
	tempFiles = map[string]struct{}{}
	tempFilesMu.Unlock()

	for _, path := range paths {
		_ = os.Remove(path)
	}
}
