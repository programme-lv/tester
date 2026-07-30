package tester

import (
	"errors"
	"testing"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/filecache"
)

func TestPrepareFileFallsBackToURL(t *testing.T) {
	store := filecache.New(t.TempDir(), t.TempDir())
	tester := &Tester{filestore: store}
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	badURL := "%"
	fetcher := &failingFetcher{}

	err := tester.prepareFile("eval", &api.File{Sha256: &key, Url: &badURL}, fetcher)
	if err == nil {
		t.Fatal("expected URL fallback error")
	}
	if !fetcher.called {
		t.Fatal("NATS fetch was not attempted")
	}
}

type failingFetcher struct {
	called bool
}

func (f *failingFetcher) Fetch(string, string) error {
	f.called = true
	return errors.New("NATS unavailable")
}
