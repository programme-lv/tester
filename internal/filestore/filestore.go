package filestore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/puzpuzpuz/xsync/v3"
)

type FileStore struct {
	fileDir string             // file directory
	tmpDir  string             // temporary directory
	await   mapset.Set[string] // awaited keys (prioritized)
	cond    *sync.Cond         // change announcements
	queue   chan string        // queue of scheduled keys
	urls    *xsync.MapOf[string, chan *url.URL]
}

func New(fileDir string, tmpDir string) *FileStore {

	err := os.MkdirAll(fileDir, 0755)
	if err != nil {
		errMsg := "failed to create directory %s: %w"
		panic(fmt.Errorf(errMsg, fileDir, err))
	}

	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		errMsg := "failed to create directory %s: %w"
		panic(fmt.Errorf(errMsg, tmpDir, err))
	}

	fs := &FileStore{
		fileDir: fileDir,
		tmpDir:  tmpDir,
		await:   mapset.NewSet[string](),
		urls:    xsync.NewMapOf[string, chan *url.URL](),
		queue:   make(chan string, 1024),
		cond:    sync.NewCond(&sync.Mutex{}),
	}

	return fs
}

// Await waits for the file to be downloaded and returns its contents.
func (fs *FileStore) Await(key string) ([]byte, error) {
	if err := validateHexSha256(key); err != nil {
		errMsg := "invalid file key %s: %w"
		return nil, fmt.Errorf(errMsg, key, err)
	}

	fs.cond.L.Lock()
	defer fs.cond.L.Unlock()

	for !fs.Exists(key) {
		urls, exists := fs.urls.Load(key)
		if !exists {
			errMsg := "file %s has not been scheduled for download"
			return nil, fmt.Errorf(errMsg, key)
		}
		if len(urls) == 0 {
			errMsg := "file %s download has been unsuccessful"
			return nil, fmt.Errorf(errMsg, key)
		}

		fs.await.Add(key)
		fs.cond.Wait()
	}

	return fs.Get(key)
}

func (fs *FileStore) Store(data []byte) (string, error) {
	// get sha256 hash of the data
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		errMsg := "failed to compute SHA256 of data: %w"
		return "", fmt.Errorf(errMsg, err)
	}
	sha256Key := hex.EncodeToString(hasher.Sum(nil))

	// save the data to the file store
	filePath := fs.path(sha256Key)
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		errMsg := "failed to write data to file %s: %w"
		return "", fmt.Errorf(errMsg, filePath, err)
	}

	return sha256Key, nil
}

func (fs *FileStore) Schedule(sha256Key string, downlUrl string) error {
	if err := validateHexSha256(sha256Key); err != nil {
		errMsg := "invalid file key %s: %w"
		return fmt.Errorf(errMsg, sha256Key, err)
	}

	parsedUrl, err := url.Parse(downlUrl)
	if err != nil {
		errMsg := "failed to parse URL %s: %w"
		return fmt.Errorf(errMsg, downlUrl, err)
	}

	fs.cond.L.Lock()
	defer fs.cond.L.Unlock()

	// if the map does not yet have a channel for this key, create one
	// and send the URL to it, otherwise just send the URL to the existing
	n := make(chan *url.URL, 4096)
	n <- parsedUrl
	c, loaded := fs.urls.LoadOrStore(sha256Key, n)
	if loaded {
		c <- parsedUrl
	}

	fs.queue <- sha256Key
	fs.cond.Broadcast()

	return nil
}

// Start starts a indefinite loop for downloading files.
// It downloads files in the order of their arrival,
// prioritizing those files that are currently awaited by the tester.
func (fs *FileStore) Start() {
	for {
		awaited := fs.await.ToSlice()
		for _, key := range awaited {
			err := fs.download(key)
			if err != nil {
				errMsg := "failed to download file %s: %v"
				log.Printf(errMsg, key, err)
				continue
			}
		}
		// choose some random key from not awaited
		key := <-fs.queue
		err := fs.download(key)
		if err != nil {
			errMsg := "failed to download file %s: %v"
			log.Printf(errMsg, key, err)
		}
	}
}

// Ensures file exists on its path or otherwise downloads it.
// Loads the condition variable and announces that the file exists.
func (fs *FileStore) download(key string) error {
	fs.cond.L.Lock()
	defer fs.cond.L.Unlock()

	urls, exists := fs.urls.Load(key)
	if !exists {
		errMsg := "file %s has not been scheduled for download"
		return fmt.Errorf(errMsg, key)
	}

	urlSlice := make([]*url.URL, 0, len(urls))
	for len(urls) > 0 {
		urlSlice = append(urlSlice, <-urls)
	}

	for _, url := range urlSlice {
		if fs.Exists(key) {
			break
		}
		err := download(url.String(), fs.tmpDir, fs.path(key), key)
		if err != nil {
			errMsg := "failed to download file %s from %s: %w"
			fs.cond.Broadcast()
			return fmt.Errorf(errMsg, key, url.String(), err)
		}
		fs.await.Remove(key)
		fs.cond.Broadcast()
	}

	return nil
}

func (fs *FileStore) path(key string) string {
	return filepath.Join(fs.fileDir, key)
}

func (fs *FileStore) Get(key string) ([]byte, error) {
	filePath := fs.path(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		errMsg := "failed to read file %s: %w"
		return nil, fmt.Errorf(errMsg, key, err)
	}
	return data, nil
}

func (fs *FileStore) Exists(key string) bool {
	_, err := os.Stat(fs.path(key))
	return err == nil
}

// Downloads a file from the given URL which is likely to be an S3 presigned URL.
// If the file is compressed with zstd, as indicated by the Content-Type or ext,
// it will be decompressed before saving. URL scheme must be HTTPS.
// Adds integrity check using a provided SHA256 hash.
func download(downlURL string, tmpDir string, saveToPath string, expectedSha256 string) error {
	u, err := url.Parse(downlURL)
	if err != nil {
		errMsg := "failed to parse URL %s: %w"
		return fmt.Errorf(errMsg, downlURL, err)
	}

	if u.Scheme != "https" {
		errMsg := "invalid URL scheme: %s"
		return fmt.Errorf(errMsg, u.Scheme)
	}

	// Validate the expected SHA256 hash
	if err := validateHexSha256(expectedSha256); err != nil {
		errMsg := "invalid expected SHA256 hash %s: %w"
		return fmt.Errorf(errMsg, expectedSha256, err)
	}

	tmpFile, err := os.Create(filepath.Join(tmpDir, uuid.New().String()))
	if err != nil {
		errMsg := "failed to create temp file: %w"
		return fmt.Errorf(errMsg, err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name()) // Clean up temp file in case of failure
	}()

	log.Printf("Downloading file from %s to temporary path %s", downlURL, tmpFile.Name())
	resp, err := http.Get(downlURL)
	if err != nil {
		errMsg := "failed to download file from %s: %w"
		return fmt.Errorf(errMsg, downlURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := "unexpected status code %d while downloading file from %s"
		return fmt.Errorf(errMsg, resp.StatusCode, downlURL)
	}

	if (resp.Header.Get("Content-Type") == "application/zstd") ||
		filepath.Ext(u.Path) == ".zst" {

		d, err := zstd.NewReader(resp.Body)
		if err != nil {
			errMsg := "failed to create zstd reader: %w"
			return fmt.Errorf(errMsg, err)
		}
		defer d.Close()

		_, err = io.Copy(tmpFile, d)
		if err != nil {
			errMsg := "failed to write decompressed file to %s: %w"
			return fmt.Errorf(errMsg, tmpFile.Name(), err)
		}

	} else {

		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			errMsg := "failed to write file to %s: %w"
			return fmt.Errorf(errMsg, tmpFile.Name(), err)
		}

	}

	// Ensure all writes to the temp file are flushed
	if err := tmpFile.Sync(); err != nil {
		errMsg := "failed to sync temp file %s: %w"
		return fmt.Errorf(errMsg, tmpFile.Name(), err)
	}

	// Compute SHA256 of the downloaded file
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		errMsg := "failed to seek to start of temp file %s: %w"
		return fmt.Errorf(errMsg, tmpFile.Name(), err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, tmpFile); err != nil {
		errMsg := "failed to compute SHA256 of temp file %s: %w"
		return fmt.Errorf(errMsg, tmpFile.Name(), err)
	}
	computedHash := hex.EncodeToString(hasher.Sum(nil))
	if computedHash != expectedSha256 {
		errMsg := "SHA256 mismatch for file %s: expected %s, got %s"
		return fmt.Errorf(errMsg, saveToPath, expectedSha256, computedHash)
	}

	// Rename the temporary file to the target path atomically
	if err := os.Rename(tmpFile.Name(), saveToPath); err != nil {
		errMsg := "failed to rename temp file %s to %s: %w"
		return fmt.Errorf(errMsg, tmpFile.Name(), saveToPath, err)
	}

	log.Printf("Successfully downloaded and moved file to %s", saveToPath)
	return nil
}

func validateHexSha256(key string) error {
	if len(key) != 64 {
		errMsg := "invalid key length: expected 64 characters, got %d"
		return fmt.Errorf(errMsg, len(key))
	}
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			errMsg := "invalid character in key: %c. allowed: 0-9, a-f"
			return fmt.Errorf(errMsg, c)
		}
	}
	return nil
}
