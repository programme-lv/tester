package tester

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/filecache"
	"github.com/programme-lv/tester/internal/gatherer/respbuilder"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/testlib"
)

// gatedFetcher serves configured contents and blocks configured keys until
// released or cancelled, so tests can observe which files are in flight.
type gatedFetcher struct {
	store   *filecache.FileStore
	ready   map[string]string
	started chan string
	release <-chan struct{}
	blocked string

	mu        sync.Mutex
	calls     map[string]int
	active    int
	maxActive int
}

func newGatedFetcher(store *filecache.FileStore, ready map[string]string) *gatedFetcher {
	return &gatedFetcher{
		store:   store,
		ready:   ready,
		started: make(chan string, 32),
		calls:   make(map[string]int),
	}
}

func (g *gatedFetcher) Fetch(id, key string) error {
	return g.FetchContext(context.Background(), id, key)
}

func (g *gatedFetcher) FetchContext(ctx context.Context, _, key string) error {
	g.mu.Lock()
	g.calls[key]++
	g.active++
	g.maxActive = max(g.maxActive, g.active)
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.active--
		g.mu.Unlock()
	}()

	g.started <- key
	if key == g.blocked {
		select {
		case <-g.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if data, ok := g.ready[key]; ok {
		_, err := g.store.Store([]byte(data))
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

type progressGatherer struct {
	*respbuilder.Builder
	first chan struct{}
}

func (g *progressGatherer) FinishTest(id int64, subm, checker *api.RuntimeData) {
	g.Builder.FinishTest(id, subm, checker)
	if id == 1 {
		close(g.first)
	}
}

func remoteFile(data string) api.File {
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
	url := "https://example.invalid/" + key
	return api.File{Sha256: &key, Url: &url}
}

func TestColdCacheEarlyFilesAvailableWhileLaterDownloadsBlocked(t *testing.T) {
	store := filecache.New(t.TempDir(), t.TempDir())
	in, ans, later := remoteFile("input"), remoteFile("answer"), remoteFile("later")
	g := newGatedFetcher(store, map[string]string{*in.Sha256: "input", *ans.Sha256: "answer"})
	tester := &Tester{filestore: store, logger: slog.Default()}

	f, err := tester.prefetchTests("eval", []api.Test{
		{In: in, Ans: ans},
		{In: later, Ans: ans},
	}, g)
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()

	done := make(chan error, 1)
	go func() {
		for _, file := range []api.File{in, ans} {
			if _, err := f.await(*file.Sha256); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("early test blocked on a later download")
	}
	select {
	case <-f.files[*later.Sha256].done:
		t.Fatal("later file should still be downloading")
	default:
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.calls[*ans.Sha256] != 1 {
		t.Fatalf("shared answer downloaded %d times", g.calls[*ans.Sha256])
	}
}

func TestPrefetchLimitsConcurrency(t *testing.T) {
	store := filecache.New(t.TempDir(), t.TempDir())
	g := newGatedFetcher(store, nil)
	var tests []api.Test
	for i := range 10 {
		tests = append(tests, api.Test{In: remoteFile(fmt.Sprint(i)), Ans: remoteFile("shared")})
	}
	tester := &Tester{filestore: store, logger: slog.Default()}

	f, err := tester.prefetchTests("eval", tests, g)
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	for range fileDownloadWorkers {
		select {
		case <-g.started:
		case <-time.After(3 * time.Second):
			t.Fatal("workers did not start")
		}
	}
	time.Sleep(50 * time.Millisecond) // let any extra worker pick up a job
	g.mu.Lock()
	maxActive := g.maxActive
	g.mu.Unlock()
	if maxActive != fileDownloadWorkers {
		t.Fatalf("max concurrent transfers = %d, want %d", maxActive, fileDownloadWorkers)
	}
}

func TestInlineAndWarmCacheNeedNoDownload(t *testing.T) {
	store := filecache.New(t.TempDir(), t.TempDir())
	key, err := store.Store([]byte("cached"))
	if err != nil {
		t.Fatal(err)
	}
	url, content := "https://example.invalid/", "inline"
	tests := []api.Test{{In: api.File{Content: &content}, Ans: api.File{Sha256: &key, Url: &url}}}
	g := newGatedFetcher(store, nil)
	tester := &Tester{filestore: store, logger: slog.Default()}

	f, err := tester.prefetchTests("eval", tests, g)
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	for _, file := range []api.File{tests[0].In, tests[0].Ans} {
		if _, err := f.await(*file.Sha256); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.calls) != 0 {
		t.Fatal("download on a cache hit")
	}
}

// Exercise compilation, actual solution/checker execution and result emission
// while the second input is deliberately unavailable. Only trusted test code
// runs in direct mode; production continues to use isolate.
func TestExecutionStreamsBeforeAllFilesDownloaded(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ required")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	isolate.SetDirectMode(true)
	defer isolate.SetDirectMode(false)

	store := filecache.New(t.TempDir(), t.TempDir())
	in, ans, later := remoteFile("input"), remoteFile("answer"), remoteFile("later")
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()

	g := newGatedFetcher(store, map[string]string{
		*in.Sha256: "input", *ans.Sha256: "answer", *later.Sha256: "later",
	})
	g.blocked, g.release = *later.Sha256, release
	tester := NewTester(store, testlib.NewTestlibCompiler(), "test", "")

	checker := "int main() { return 0; }"
	compile, binary := "cp solution.sh compiled.sh", "compiled.sh"
	req := api.ExecReq{
		Uuid:    "cold-cache-test",
		Code:    "cat",
		Lang:    api.PrLang{CodeFname: "solution.sh", CompileCmd: &compile, CompiledFname: &binary, ExecCmd: "sh compiled.sh"},
		Checker: &checker,
		CpuMs:   1000,
		RamKiB:  65536,
		Tests:   []api.Test{{In: in, Ans: ans}, {In: later, Ans: ans}},
	}
	results := &progressGatherer{Builder: respbuilder.New(req.Uuid), first: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- tester.ExecTestsWithFileFetcher(results, req, g) }()

	select {
	case <-results.first:
	case err := <-done:
		t.Fatalf("execution ended before first result: %v", err)
	case <-time.After(10 * time.Second):
		unblock()
		<-done
		t.Fatal("no first result while later input blocked")
	}
	unblock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	response := results.Response()
	if response.Compilation == nil || len(response.TestResults) != 2 || response.Status != api.Success {
		t.Fatalf("unexpected response: %+v", response)
	}
}
