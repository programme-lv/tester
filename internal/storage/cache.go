package storage

import (
	"os"
	"path"
	"path/filepath"

	"github.com/programme-lv/tester/internal/database"
)

var textFileCachePath string = path.Join(GetCacheDir(), "text_files")

func GetCacheDir() string {
	return path.Join(MustGetUserHomeDir(), ".cache", "tester")
}

func MustGetUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return home
}

func SaveTextFileToCache(textFile *database.TextFile) error {
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

func IsTextFileInCache(sha256 string) (bool, error) {
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
