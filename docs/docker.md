# Docker verification

Build from the tester repository root; no separate isolate checkout is needed:

```sh
docker build -t tester:dev .
docker run --rm --privileged --cgroupns=private tester:dev
```

This runs `tester verify ./behaviour.toml` using a copy of `docs/behave.toml`.
The image includes isolate v2.7, C++17, Python 3.13, Go 1.25, Java 25,
and testlib.h. System information is a container placeholder.

Use a Linux Docker host with cgroup v2 and a writable, private cgroup mount.
The entrypoint moves container processes into a leaf cgroup and enables the
cpuset and memory controllers for isolate without systemd. Do not bind-mount
the host's `/sys/fs/cgroup`: setup assumes the mount is scoped to this container.
The current unprivileged devbox cannot launch isolate itself; run this image
as a sibling container from the host.

For verbose verification:

```sh
docker run --rm --privileged --cgroupns=private tester:dev \
  tester verify --verbose ./behaviour.toml
```

## File-locking restriction

The image sets isolate's `syscall_flags = 65531` (`65535 & ~4`), allowing the
file-locking operations needed by Go's build-cache cleanup. This applies to
all sandboxes, including submitted programs, and permits shared-file locking
as a communication channel. This image reproduces the development verification
configuration. Restricting the exception to Go compilation would require a
separate tester change; it is not implemented here.

## Devbox Compose

The host-side `/workspace/meta/compose.yaml.draft` adds an optional `tester`
profile with this repository as its build context. After reviewing and copying
the draft to the host Compose directory as `compose.yaml`, run there:

```sh
docker compose --profile tester run --build --rm tester
```

To run a worker on the same network as the shared NATS service:

```sh
docker compose up -d nats
docker compose --profile tester run --build --rm tester tester listen nats
```

The draft sets `NATS_URL=nats://nats:4222`. Verification itself does not need NATS.
