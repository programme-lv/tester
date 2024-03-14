package storage

import (
	"os"
	"path"
)

func getUserCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(home, ".cache", "tester"), nil
}

func (s *Storage) textFileCachePath() string {
	return path.Join(s.dir, "text_files")
}

func (s *Storage) testlibCachePath() string {
	return path.Join(s.dir, "testlib.h")
}
