package filestore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	fileDirectory    string
	s3DownloadFunc   func(s3Uri string) (string, error)
	downloadChannels sync.Map
	awaitedKeyQueue  chan string
	scheduledS3Files chan string
	fileKeyToS3Uri   sync.Map
}

// NewFileStore creates a new FileStore instance. It takes a function that downloads files from S3.
func NewFileStore(downloadFunc func(s3Uri string) (string, error)) *FileStore {
	fs := &FileStore{
		fileDirectory:    filepath.Join("var", "tester", "files"),
		s3DownloadFunc:   downloadFunc,
		downloadChannels: sync.Map{},
		awaitedKeyQueue:  make(chan string, 10000),
		scheduledS3Files: make(chan string, 10000),
		fileKeyToS3Uri:   sync.Map{},
	}

	err := os.MkdirAll(fs.fileDirectory, 0777)
	if err != nil {
		panic(fmt.Errorf("failed to create file store directory: %w", err))
	}

	return fs
}

// AwaitAndGetFile waits for the download to finish (if it hasn't already), and then returns the file's contents.
func (fs *FileStore) AwaitAndGetFile(key string) ([]byte, error) {
	downlChan, exists := fs.downloadChannels.Load(key)
	if !exists {
		return nil, fmt.Errorf("file %s has not been added to file store", key)
	}
	fs.awaitedKeyQueue <- key
	<-downlChan.(chan struct{})

	filePath := filepath.Join(fs.fileDirectory, key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", key, err)
	}
	return data, nil
}

// ScheduleDownloadFromS3 schedules a download from S3 if it's not already in progress or completed.
func (fs *FileStore) ScheduleDownloadFromS3(key string, s3Uri string) error {
	_, loaded := fs.fileKeyToS3Uri.LoadOrStore(key, s3Uri)
	if loaded {
		return nil // already scheduled
	}

	c := make(chan struct{}, 1)
	_, loaded = fs.downloadChannels.LoadOrStore(key, c)
	if loaded {
		return nil // already scheduled
	}

	fs.scheduledS3Files <- key
	return nil
}

func (fs *FileStore) StartDownloadingInBg() {
	// download files in bacgkround, prioritize awaited files
	go func() {
		for {
			// if both are ready, prioritize awaited files
			for len(fs.awaitedKeyQueue) > 0 {
				err := fs.download(<-fs.awaitedKeyQueue)
				if err != nil {
					fmt.Printf("failed to download file: %v", err)
					panic(err)
				}
			}
			err := fs.download(<-fs.scheduledS3Files)
			if err != nil {
				fmt.Printf("failed to download file: %v", err)
				panic(err)
			}
		}
	}()
}

func (fs *FileStore) download(key string) error {
	_, err := os.Stat(filepath.Join(fs.fileDirectory, key))
	if err == nil {
		return nil
	}

	s3Uri, exists := fs.fileKeyToS3Uri.Load(key)
	if !exists {
		return fmt.Errorf("file %s has not been scheduled for download", key)
	}
	tmpPath, err := fs.s3DownloadFunc(s3Uri.(string))
	if err != nil {
		return fmt.Errorf("failed to download file %s from S3: %w", key, err)
	}
	filePath := filepath.Join(fs.fileDirectory, key)
	err = os.Rename(tmpPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to move file %s to file store: %w", key, err)
	}
	return nil
}
