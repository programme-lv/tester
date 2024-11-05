package filestore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	fileDirectory    string
	tmpDirectory     string
	downloadFunc     func(s3Url string, path string) error
	awaitedKeyQueue  chan string
	scheduledS3Files chan string
	fileKeyToS3Url   *sync.Map
	downloadLocks    *sync.Map
	downloadedSet    *sync.Map
}

// NewFileStore creates a new FileStore instance. It takes a function that downloads files from S3.
func NewFileStore(downloadFunc func(s3Url string, path string) error) *FileStore {
	fs := &FileStore{
		fileDirectory:    filepath.Join("var", "tester", "files"),
		tmpDirectory:     filepath.Join("var", "tester", "tmp"),
		downloadFunc:     downloadFunc,
		awaitedKeyQueue:  make(chan string, 10000),
		scheduledS3Files: make(chan string, 10000),
		fileKeyToS3Url:   &sync.Map{},
		downloadLocks:    &sync.Map{},
		downloadedSet:    &sync.Map{},
	}

	err := os.MkdirAll(fs.fileDirectory, 0777)
	if err != nil {
		panic(fmt.Errorf("failed to create file store directory: %w", err))
	}

	err = os.MkdirAll(fs.tmpDirectory, 0777)
	if err != nil {
		panic(fmt.Errorf("failed to create tmp directory: %w", err))
	}

	return fs
}

// AwaitAndGetFile waits for the download to finish (if it hasn't already), and then returns the file's contents.
func (fs *FileStore) AwaitAndGetFile(key string) ([]byte, error) {
	fs.awaitedKeyQueue <- key

	lock, exists := fs.downloadLocks.Load(key)
	if !exists {
		return nil, fmt.Errorf("file %s has not been scheduled for download", key)
	}

	downloadExists := false
	_, downloadExists = fs.downloadedSet.Load(key)
	for !downloadExists {
		lock.(*sync.Cond).Wait()
		_, downloadExists = fs.downloadedSet.Load(key)
	}

	filePath := filepath.Join(fs.fileDirectory, key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", key, err)
	}
	return data, nil
}

// ScheduleDownloadFromS3 schedules a download from S3 if it's not already in progress or completed.
func (fs *FileStore) ScheduleDownloadFromS3(key string, s3Url string) error {
	_, loaded := fs.fileKeyToS3Url.LoadOrStore(key, s3Url)
	if loaded {
		return nil // already scheduled
	}

	lock := sync.NewCond(&sync.Mutex{})
	lock.L.Lock()
	fs.downloadLocks.Store(key, lock)

	fs.scheduledS3Files <- key

	return nil
}

func (fs *FileStore) StartDownloadingInBg() {
	// download files in bacgkround, prioritize awaited files
	go func() {
		for {
			var key string
			select {
			case key = <-fs.awaitedKeyQueue:
			case key = <-fs.scheduledS3Files:
			}
			err := fs.downloadIfDoesNotExist(key)
			if err != nil {
				fmt.Printf("failed to download file: %v", err)
				panic(err)
			}
		}
	}()
}

func (fs *FileStore) downloadIfDoesNotExist(key string) error {
	lock, lockExists := fs.downloadLocks.Load(key)
	if !lockExists {
		return fmt.Errorf("lock does not exist for file %s", key)
	}

	_, err := os.Stat(filepath.Join(fs.fileDirectory, key))
	if err == nil {
		fs.downloadedSet.Store(key, struct{}{})
		lock.(*sync.Cond).Broadcast()
		return nil
	}

	s3Url, exists := fs.fileKeyToS3Url.Load(key)
	if !exists {
		return fmt.Errorf("file %s has not been scheduled for download", key)
	}
	tmpPath := filepath.Join(fs.tmpDirectory, key)
	err = fs.downloadFunc(s3Url.(string), tmpPath)
	if err != nil {
		return fmt.Errorf("failed to download file %s from S3: %w", key, err)
	}
	filePath := filepath.Join(fs.fileDirectory, key)
	err = os.Rename(tmpPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to move file %s to file store: %w", key, err)
	}

	fs.downloadedSet.Store(key, struct{}{})
	lock.(*sync.Cond).Broadcast()
	return nil
}
