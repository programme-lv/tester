package filecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestNewSetsHTTPTimeout(t *testing.T) {
	store := New(t.TempDir(), t.TempDir())
	if store.client.Timeout != httpTimeout {
		t.Fatalf("timeout is %s", store.client.Timeout)
	}
}

func TestDownloadURLStoresPlainAndZstd(t *testing.T) {
	data := []byte("test file contents")
	compressed := compress(t, data)
	tests := []struct {
		name, path, contentType string
		body                    []byte
	}{
		{"plain", "/file", "text/plain", data},
		{"zstd", "/file.zst", "application/zstd", compressed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, dir, baseURL := testStore(t, tt.path, tt.contentType, tt.body)
			if err := store.downloadURL(baseURL+tt.path, hash(data)); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, hash(data)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestDownloadURLFailureCleansTemp(t *testing.T) {
	data := []byte("test file contents")
	compressed := compress(t, data)
	tests := []struct {
		name, path, contentType string
		body                    []byte
		key                     string
		maxCompressed           int64
		maxDecompressed         int64
	}{
		{"hash", "/file", "text/plain", data, hash([]byte("other")), 1 << 20, 1 << 20},
		{"plain size", "/file", "text/plain", data, hash(data), 2, 1 << 20},
		{"compressed size", "/file.zst", "application/zstd", compressed, hash(data), int64(len(compressed) - 1), 1 << 20},
		{"decompressed size", "/file.zst", "application/zstd", compressed, hash(data), 1 << 20, 2},
		{"invalid zstd", "/file.zst", "application/zstd", data, hash(data), 1 << 20, 1 << 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, dir, baseURL := testStore(t, tt.path, tt.contentType, tt.body)
			err := store.downloadURLWithLimits(
				baseURL+tt.path,
				tt.key,
				tt.maxCompressed,
				tt.maxDecompressed,
			)
			if err == nil {
				t.Fatal("expected error")
			}
			assertEmpty(t, dir)
		})
	}
}

func testStore(t *testing.T, path, contentType string, body []byte) (*FileStore, string, string) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	store := New(dir, t.TempDir())
	store.client = server.Client()
	store.client.Timeout = httpTimeout
	return store, dir, server.URL
}

func compress(t *testing.T, data []byte) []byte {
	t.Helper()
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil)
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("cache contains %v", entries)
	}
}
