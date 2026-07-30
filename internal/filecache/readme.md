# Test-file cache

`FileStore` stores decompressed test files by the SHA-256 hash of their contents.
`Schedule` queues an HTTPS download for a known hash, and `Await` waits until the file exists or all scheduled retrieval attempts fail.
Downloads use temporary sibling files in the cache directory so the verified rename is atomic.
Plain and zstd responses are size-limited and hash-verified before the temporary file becomes a cache entry.

The NATS listener can populate the same cache before falling back to HTTPS.
See [NATS test-file retrieval](../../docs/nats-test-files.md) for the transfer protocol and lifecycle.

The cache currently does not implement per-user accounting or disk-usage eviction.