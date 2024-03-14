package storage

import (
	"os"
	"path/filepath"
)

func (s *Storage) SaveTextFileToCache(fname string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.MkdirAll(s.textFileCachePath(), 0755)
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.textFileCachePath(), fname)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_, err = file.Write(content)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) IsTextFileInCache(fname string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.textFileCachePath(), fname)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
