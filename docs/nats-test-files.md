# NATS test-file retrieval

The NATS listener can retrieve missing test files through the backend that submitted the job.
This path uses Core NATS and does not require JetStream or Object Store.
NATS is only a transport; the tester cache and backend file store own the file data.

## Retrieval

The backend includes its file-service subject in the `Proglv-File-Subject` job header.
The tester checks its SHA-256 cache before making a request.
For every cache miss, it creates a temporary reply inbox and requests the file with the execution UUID and expected SHA-256 hash.

The backend streams the stored zstd object as raw chunks.
`Proglv-File-Status` identifies `chunk`, `done`, and `error` messages.
Chunk messages include a zero-based `Proglv-File-Sequence`.
Error messages include `Proglv-File-Error`.

The tester rejects missing or out-of-order frames, oversized chunks, transfer timeouts, and streams exceeding the compressed-size limit.
It decompresses into a temporary file with a decompressed-size limit, verifies the expected SHA-256 hash, syncs the file, and atomically moves it into the cache.
Partial and invalid files are removed.
The limits are 512 KiB per chunk, 50 MiB compressed, 50 MiB decompressed, 5 seconds between chunks, and 30 seconds for the complete transfer.

If NATS retrieval fails, the tester schedules the signed HTTPS URL already present in the execution request.
Jobs without the file-service header use HTTPS directly.
The SQS listener is unchanged.

## Lifecycle

The reply inbox exists only for one file transfer and is unsubscribed on every exit path.
Core NATS does not persist or replay the request or chunks.
A disconnected or interrupted transfer must start again from the first chunk.

The tester's decompressed file cache remains under `$XDG_CACHE_HOME/tester/files`.
Incoming files use temporary sibling files on the cache filesystem so the verified rename is atomic.
They never become named cache entries until verification succeeds.
