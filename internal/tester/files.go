package tester

import (
	"context"
	"fmt"
	"sync"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/filecache"
)

const fileDownloadWorkers = 4

type pendingFile struct {
	file api.File
	done chan struct{}
	err  error // published by closing done
}

type testFiles struct {
	store  *filecache.FileStore
	files  map[string]*pendingFile
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Register all files before starting workers. Inline contents are normalized here
// so workers never mutate the request while the test loop is reading it.
func (t *Tester) prefetchTests(evalUUID string, tests []api.Test, fetcher FileFetcher) (*testFiles, error) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &testFiles{store: t.filestore, files: make(map[string]*pendingFile), cancel: cancel}
	var queue []*pendingFile
	for i := range tests {
		for _, file := range []*api.File{&tests[i].In, &tests[i].Ans} {
			if file.Content != nil {
				key, err := t.filestore.Store([]byte(*file.Content))
				if err != nil {
					cancel()
					return nil, fmt.Errorf("store test %d content: %w", i+1, err)
				}
				file.Sha256 = &key
			}
			if file.Sha256 == nil || (file.Content == nil && file.Url == nil) {
				cancel()
				return nil, fmt.Errorf("test %d: missing file hash or source", i+1)
			}
			key := *file.Sha256
			if _, exists := f.files[key]; exists {
				continue
			}
			p := &pendingFile{file: *file, done: make(chan struct{})}
			f.files[key] = p
			if f.store.Exists(key) {
				close(p.done)
			} else {
				queue = append(queue, p)
			}
		}
	}
	// FIFO submission gives early tests priority without starting a goroutine
	// per file. Hash deduplication also handles shared answers and inputs.
	jobs := make(chan *pendingFile, len(queue))
	for _, p := range queue {
		jobs <- p
	}
	close(jobs)
	for range min(fileDownloadWorkers, len(queue)) {
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			for p := range jobs {
				if ctx.Err() != nil {
					return
				}
				p.err = t.fetchFile(ctx, evalUUID, p.file, fetcher)
				close(p.done)
			}
		}()
	}
	return f, nil
}

func (t *Tester) fetchFile(ctx context.Context, evalUUID string, file api.File, fetcher FileFetcher) error {
	key := *file.Sha256
	if t.filestore.Exists(key) {
		return nil
	}
	if fetcher != nil {
		var err error
		if cf, ok := fetcher.(interface {
			FetchContext(context.Context, string, string) error
		}); ok {
			err = cf.FetchContext(ctx, evalUUID, key)
		} else {
			err = fetcher.Fetch(evalUUID, key)
		}
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		t.logger.Warn("NATS test file fetch failed; falling back to HTTPS", "sha256", key, "error", err)
	}
	return t.filestore.DownloadContext(ctx, *file.Url, key)
}

func (f *testFiles) await(key string) ([]byte, error) {
	p, ok := f.files[key]
	if !ok {
		return nil, fmt.Errorf("test file %s not registered", key)
	}
	<-p.done
	if p.err != nil {
		return nil, p.err
	}
	return f.store.Get(key)
}

func (f *testFiles) close() {
	f.cancel()
	f.wg.Wait()
}
