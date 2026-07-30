package natsfile

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	SubjectHeader  = "Proglv-File-Subject"
	StatusHeader   = "Proglv-File-Status"
	SequenceHeader = "Proglv-File-Sequence"
	ErrorHeader    = "Proglv-File-Error"
)

type Cache interface {
	Exists(string) bool
	StoreZstd(string, io.Reader, int64, int64) error
}

type Options struct {
	ChunkTimeout         time.Duration
	TransferTimeout      time.Duration
	MaxChunkBytes        int64
	MaxCompressedBytes   int64
	MaxDecompressedBytes int64
}

func DefaultOptions() Options {
	return Options{
		ChunkTimeout:         5 * time.Second,
		TransferTimeout:      30 * time.Second,
		MaxChunkBytes:        512 << 10,
		MaxCompressedBytes:   50 << 20,
		MaxDecompressedBytes: 50 << 20,
	}
}

type Fetcher struct {
	b       broker
	cache   Cache
	subject string
	opts    Options
}

func New(nc *nats.Conn, cache Cache, subject string) *Fetcher {
	return newFetcher(natsBroker{nc}, cache, subject, DefaultOptions())
}

func newFetcher(b broker, cache Cache, subject string, opts Options) *Fetcher {
	return &Fetcher{b: b, cache: cache, subject: subject, opts: opts}
}

func (f *Fetcher) Fetch(evalUUID, sha256 string) error {
	if f.cache.Exists(sha256) {
		return nil
	}
	sub, reply, err := f.subscribe()
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	body, err := json.Marshal(request{EvalUUID: evalUUID, SHA256: sha256})
	if err != nil {
		return fmt.Errorf("marshal file request: %w", err)
	}
	if err := f.b.publishRequest(f.subject, reply, body); err != nil {
		return fmt.Errorf("publish file request: %w", err)
	}
	if err := f.b.flush(); err != nil {
		return fmt.Errorf("flush file request: %w", err)
	}
	return f.receive(sub, sha256)
}

func (f *Fetcher) subscribe() (subscription, string, error) {
	reply := f.b.newInbox()
	sub, err := f.b.subscribeSync(reply)
	if err != nil {
		return nil, "", fmt.Errorf("subscribe file reply: %w", err)
	}
	return sub, reply, nil
}

func (f *Fetcher) receive(sub subscription, sha256 string) error {
	reader, writer := io.Pipe()
	stored := make(chan error, 1)
	go func() {
		err := f.cache.StoreZstd(
			sha256,
			reader,
			f.opts.MaxCompressedBytes,
			f.opts.MaxDecompressedBytes,
		)
		_ = reader.CloseWithError(err)
		stored <- err
	}()

	err := f.receiveChunks(sub, writer)
	_ = writer.CloseWithError(err)
	storeErr := <-stored
	if err != nil {
		return err
	}
	if storeErr != nil {
		return fmt.Errorf("store NATS file: %w", storeErr)
	}
	return nil
}

func (f *Fetcher) receiveChunks(sub subscription, dst *io.PipeWriter) error {
	deadline := time.Now().Add(f.opts.TransferTimeout)
	var sequence, total int64
	for {
		if time.Until(deadline) <= 0 {
			return fmt.Errorf("file transfer timeout")
		}
		msg, err := sub.NextMsg(nextTimeout(deadline, f.opts.ChunkTimeout))
		if err != nil {
			return fmt.Errorf("receive file chunk: %w", err)
		}
		switch msg.Header.Get(StatusHeader) {
		case "chunk":
			if err := f.writeChunk(dst, msg, sequence, &total); err != nil {
				return err
			}
			sequence++
		case "done":
			return nil
		case "error":
			return fmt.Errorf("file service: %s", msg.Header.Get(ErrorHeader))
		default:
			return fmt.Errorf("invalid file status %q", msg.Header.Get(StatusHeader))
		}
	}
}

func (f *Fetcher) writeChunk(dst io.Writer, msg *nats.Msg, expected int64, total *int64) error {
	sequence, err := strconv.ParseInt(msg.Header.Get(SequenceHeader), 10, 64)
	if err != nil || sequence != expected {
		return fmt.Errorf("invalid file sequence %q, expected %d", msg.Header.Get(SequenceHeader), expected)
	}
	if int64(len(msg.Data)) > f.opts.MaxChunkBytes {
		return fmt.Errorf("file chunk exceeds %d bytes", f.opts.MaxChunkBytes)
	}
	*total += int64(len(msg.Data))
	if *total > f.opts.MaxCompressedBytes {
		return fmt.Errorf("compressed file exceeds %d bytes", f.opts.MaxCompressedBytes)
	}
	if _, err := dst.Write(msg.Data); err != nil {
		return fmt.Errorf("stream file chunk: %w", err)
	}
	return nil
}

func nextTimeout(deadline time.Time, chunkTimeout time.Duration) time.Duration {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return time.Nanosecond
	}
	if remaining < chunkTimeout {
		return remaining
	}
	return chunkTimeout
}

type request struct {
	EvalUUID string `json:"eval_uuid"`
	SHA256   string `json:"sha256"`
}
