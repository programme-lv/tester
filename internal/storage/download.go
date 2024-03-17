package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Storage) DownloadTextFile(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure the directory exists
	dirPath := s.textFileCachePath()
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create a temporary file
	tempFile, err := os.CreateTemp(s.textFileCachePath(), "download-*.tmp")
	if err != nil {
		return err
	}
	tempFilePath := tempFile.Name()
	defer tempFile.Close()

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return err
	}

	// It's a good practice to check the HTTP response status before proceeding.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Calculate SHA256 hash of the file content
	if _, err := tempFile.Seek(0, io.SeekStart); err != nil { // Rewind to start of the file
		return err
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, tempFile); err != nil {
		return err
	}
	hashValue := hex.EncodeToString(hasher.Sum(nil))

	// Close the temporary file before renaming to avoid issues on some platforms
	tempFile.Close()

	// Rename the file to its SHA256 hash
	newPath := filepath.Join(s.textFileCachePath(), hashValue)
	if err := os.Rename(tempFilePath, newPath); err != nil {
		return err
	}

	return nil
}
