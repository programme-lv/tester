package storage

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Storage) DownloadTextFile(fname string, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.textFileCachePath(), fname)
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
