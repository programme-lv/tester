package testing

import (
	"fmt"
	"github.com/programme-lv/tester/internal/database"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func saveTextFileToCache(textFile *database.TextFile) error {
	err := os.MkdirAll(textFileCachePath, 0755)
	if err != nil {
		return err
	}

	fileName := textFile.Sha256
	filePath := filepath.Join(textFileCachePath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_, err = file.Write([]byte(textFile.Content))
	if err != nil {
		return err
	}
	return nil
}

func isTextFileInCache(sha256 string) (bool, error) {
	fileName := sha256
	filePath := filepath.Join(textFileCachePath, fileName)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ensureTestlibExistsInCache() error {
	// Check if the file already exists
	if _, err := os.Stat(testLibCachePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err // Return here on other errors besides "not exists"
	}

	// Create the file
	out, err := os.Create(testLibCachePath)
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
