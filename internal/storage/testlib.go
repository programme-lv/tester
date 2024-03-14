package storage

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const testLibUrl = "https://raw.githubusercontent.com/MikeMirzayanov/testlib/master/testlib.h"

func (s *Storage) GetTestlib() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureTestlibExistsInCache(); err != nil {
		return nil, fmt.Errorf("failed to ensure testlib exists in cache: %w", err)
	}

	testlibBytes, err := os.ReadFile(s.testlibCachePath())
	if err != nil {
		return nil, fmt.Errorf("failed to read testlib from cache: %w", err)
	}

	return testlibBytes, nil
}

func (s *Storage) ensureTestlibExistsInCache() error {
	path := s.testlibCachePath()

	if _, err := os.Stat(path); err == nil {
		return nil // testlib already exists in cache
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if testlib exists in cache: %w", err)
	}

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	resp, err := http.Get(testLibUrl)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
