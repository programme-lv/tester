package natsfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/nats-io/nats.go"
	"github.com/programme-lv/tester/internal/filecache"
)

func TestFetchCacheHit(t *testing.T) {
	cache, _ := testCache(t)
	data := []byte("cached")
	key := hash(data)
	if _, err := cache.Store(data); err != nil {
		t.Fatal(err)
	}
	b := &fakeBroker{}

	if err := newFetcher(b, cache, "files", testOptions()).Fetch("eval", key); err != nil {
		t.Fatal(err)
	}
	if b.subscribed {
		t.Fatal("cache hit subscribed")
	}
}

func TestFetchMultiChunk(t *testing.T) {
	cache, fileDir := testCache(t)
	data := []byte("multi-chunk test contents")
	compressed := compress(t, data)
	mid := len(compressed) / 2
	b := brokerWith(
		chunk(0, compressed[:mid]),
		chunk(1, compressed[mid:]),
		status("done"),
	)

	if err := newFetcher(b, cache, "files", testOptions()).Fetch("eval", hash(data)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(fileDir, hash(data)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q", got)
	}
	var req request
	if err := json.Unmarshal(b.data, &req); err != nil {
		t.Fatal(err)
	}
	if req.EvalUUID != "eval" || req.SHA256 != hash(data) || !b.sub.unsubscribed {
		t.Fatalf("bad request or subscription cleanup: %+v", req)
	}
}

func TestFetchFailuresCleanUp(t *testing.T) {
	data := []byte("valid contents")
	compressed := compress(t, data)
	tests := []struct {
		name string
		msgs []*nats.Msg
		key  string
		opts Options
	}{
		{"sequence", []*nats.Msg{chunk(1, compressed)}, hash(data), testOptions()},
		{"service error", []*nats.Msg{serviceError("denied")}, hash(data), testOptions()},
		{"hash", []*nats.Msg{chunk(0, compressed), status("done")}, hash([]byte("other")), testOptions()},
		{"chunk size", []*nats.Msg{chunk(0, compressed)}, hash(data), optionsWithLimits(1, 1<<20, 1<<20)},
		{"compressed size", []*nats.Msg{chunk(0, compressed)}, hash(data), optionsWithLimits(1<<20, 1, 1<<20)},
		{"decompressed size", []*nats.Msg{chunk(0, compressed), status("done")}, hash(data), optionsWithLimits(1<<20, 1<<20, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, fileDir := testCache(t)
			b := brokerWith(tt.msgs...)
			err := newFetcher(b, cache, "files", tt.opts).Fetch("eval", tt.key)
			if err == nil {
				t.Fatal("expected error")
			}
			assertEmpty(t, fileDir)
			if !b.sub.unsubscribed {
				t.Fatal("subscription not removed")
			}
		})
	}
}

func testCache(t *testing.T) (*filecache.FileStore, string) {
	t.Helper()
	fileDir := t.TempDir()
	return filecache.New(fileDir, t.TempDir()), fileDir
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

func chunk(sequence int, data []byte) *nats.Msg {
	msg := status("chunk")
	msg.Header.Set(SequenceHeader, strconv.Itoa(sequence))
	msg.Data = data
	return msg
}

func status(value string) *nats.Msg {
	msg := nats.NewMsg("reply")
	msg.Header.Set(StatusHeader, value)
	return msg
}

func serviceError(value string) *nats.Msg {
	msg := status("error")
	msg.Header.Set(ErrorHeader, value)
	return msg
}

func testOptions() Options {
	return optionsWithLimits(1<<20, 1<<20, 1<<20)
}

func optionsWithLimits(chunk, compressed, decompressed int64) Options {
	return Options{
		ChunkTimeout:         time.Second,
		TransferTimeout:      time.Second,
		MaxChunkBytes:        chunk,
		MaxCompressedBytes:   compressed,
		MaxDecompressedBytes: decompressed,
	}
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

type fakeSubscription struct {
	msgs         []*nats.Msg
	unsubscribed bool
}

func (s *fakeSubscription) NextMsg(time.Duration) (*nats.Msg, error) {
	if len(s.msgs) == 0 {
		return nil, nats.ErrTimeout
	}
	msg := s.msgs[0]
	s.msgs = s.msgs[1:]
	return msg, nil
}

func (s *fakeSubscription) Unsubscribe() error {
	s.unsubscribed = true
	return nil
}

type fakeBroker struct {
	sub        *fakeSubscription
	subscribed bool
	subject    string
	reply      string
	data       []byte
}

func brokerWith(msgs ...*nats.Msg) *fakeBroker {
	return &fakeBroker{sub: &fakeSubscription{msgs: msgs}}
}

func (b *fakeBroker) newInbox() string {
	return "_INBOX.test"
}

func (b *fakeBroker) subscribeSync(string) (subscription, error) {
	b.subscribed = true
	if b.sub == nil {
		return nil, errors.New("unexpected subscription")
	}
	return b.sub, nil
}

func (b *fakeBroker) publishRequest(subject, reply string, data []byte) error {
	b.subject, b.reply, b.data = subject, reply, data
	return nil
}

func (b *fakeBroker) flush() error {
	return nil
}
