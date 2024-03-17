package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Storage) GetTextFile(fname string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureTextFileExistsInCache(fname); err != nil {
		return nil, err
	}

	textFileBytes, err := os.ReadFile(filepath.Join(s.textFileCachePath(), fname))
	if err != nil {
		return nil, err
	}

	return textFileBytes, nil
}

func (s *Storage) ensureTextFileExistsInCache(fname string) error {
	path := filepath.Join(s.textFileCachePath(), fname)

	if _, err := os.Stat(path); err == nil {
		return nil // text file already exists in cache
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if text file exists in cache: %w", err)
	}

	return fmt.Errorf("text file %s does not exist in cache", fname)
}

func (s *Storage) SaveTextFileToCache(content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.MkdirAll(s.textFileCachePath(), 0755)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(content)
	hexString := hex.EncodeToString(hash[:])
	filePath := filepath.Join(s.textFileCachePath(), hexString)
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
