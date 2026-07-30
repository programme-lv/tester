package filecache

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
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/puzpuzpuz/xsync/v3"
)

const (
	maxTransferBytes = 50 << 20
	httpTimeout      = 30 * time.Second
)

type FileStore struct {
	fileDir string // file directory
	tmpDir  string // temporary directory
	client  *http.Client
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
		client:  &http.Client{Timeout: httpTimeout},
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
		err := fs.downloadURL(url.String(), key)
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

func (fs *FileStore) StoreZstd(key string, src io.Reader, maxCompressed, maxDecompressed int64) error {
	return fs.storeEncoded(key, src, true, maxCompressed, maxDecompressed)
}

func (fs *FileStore) storePlain(key string, src io.Reader, maxCompressed, maxDecompressed int64) error {
	return fs.storeEncoded(key, src, false, maxCompressed, maxDecompressed)
}

func (fs *FileStore) storeEncoded(
	key string,
	src io.Reader,
	compressed bool,
	maxCompressed, maxDecompressed int64,
) error {
	if err := validateHexSha256(key); err != nil {
		return fmt.Errorf("invalid file key %s: %w", key, err)
	}
	limited := &io.LimitedReader{R: src, N: maxCompressed + 1}
	var content io.Reader = limited
	if compressed {
		decoder, err := zstd.NewReader(limited)
		if err != nil {
			return fmt.Errorf("create zstd reader: %w", err)
		}
		defer decoder.Close()
		content = decoder
	}
	validateCompressed := func() error {
		if _, err := io.Copy(io.Discard, limited); err != nil {
			return fmt.Errorf("read compressed file: %w", err)
		}
		if maxCompressed+1-limited.N > maxCompressed {
			return fmt.Errorf("compressed file exceeds %d bytes", maxCompressed)
		}
		return nil
	}
	return fs.storeVerified(key, content, maxDecompressed, validateCompressed)
}

func (fs *FileStore) storeVerified(
	key string,
	src io.Reader,
	maxBytes int64,
	validate func() error,
) error {
	tmp, err := os.CreateTemp(fs.fileDir, ".incoming-*")
	if err != nil {
		return fmt.Errorf("create cache temp file: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()
	if err := tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("set cache temp permissions: %w", err)
	}

	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(src, maxBytes+1))
	if err != nil {
		return fmt.Errorf("write cache temp file: %w", err)
	}
	if n > maxBytes {
		return fmt.Errorf("decompressed file exceeds %d bytes", maxBytes)
	}
	if hex.EncodeToString(hash.Sum(nil)) != key {
		return fmt.Errorf("SHA256 mismatch for file %s", key)
	}
	if err := validate(); err != nil {
		return err
	}
	return fs.commitTemp(tmp, key)
}

func (fs *FileStore) commitTemp(tmp *os.File, key string) error {
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync cache temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cache temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), fs.path(key)); err != nil {
		return fmt.Errorf("rename cache temp file: %w", err)
	}
	return nil
}

func (fs *FileStore) downloadURL(downlURL, expectedSHA256 string) error {
	return fs.downloadURLWithLimits(downlURL, expectedSHA256, maxTransferBytes, maxTransferBytes)
}

func (fs *FileStore) downloadURLWithLimits(
	downlURL, expectedSHA256 string,
	maxCompressed, maxDecompressed int64,
) error {
	u, err := url.Parse(downlURL)
	if err != nil {
		return fmt.Errorf("parse URL %s: %w", downlURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}
	if err := validateHexSha256(expectedSHA256); err != nil {
		return fmt.Errorf("invalid expected SHA256 hash %s: %w", expectedSHA256, err)
	}
	resp, err := fs.client.Get(downlURL)
	if err != nil {
		return fmt.Errorf("download file from %s: %w", downlURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, downlURL)
	}
	if resp.ContentLength > maxCompressed {
		return fmt.Errorf("compressed file exceeds %d bytes", maxCompressed)
	}
	if resp.Header.Get("Content-Type") == "application/zstd" || filepath.Ext(u.Path) == ".zst" {
		return fs.StoreZstd(expectedSHA256, resp.Body, maxCompressed, maxDecompressed)
	}
	return fs.storePlain(expectedSHA256, resp.Body, maxCompressed, maxDecompressed)
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
