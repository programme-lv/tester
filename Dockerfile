FROM ubuntu:25.10@sha256:7cc5e35f6567ee8c66d2abb4aab0fd866669e6207c237c3a8f0947a5c7f17092

ENV DEBIAN_FRONTEND=noninteractive
# Compilers remain installed because verification compiles submissions at runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config libcap-dev libseccomp-dev libsystemd-dev \
    ca-certificates curl golang-1.25-go python3.13 openjdk-25-jdk-headless \
    && rm -rf /var/lib/apt/lists/*
ENV PATH=/usr/lib/go-1.25/bin:$PATH

# Isolate v2.7, pinned independently of the moving tag.
ARG ISOLATE_COMMIT=8f185bb37f3f23e29b33b0c7727c91c13429abe3
RUN mkdir -p /src/isolate \
    && curl -fsSL "https://codeload.github.com/ioi/isolate/tar.gz/${ISOLATE_COMMIT}" \
       | tar -xz --strip-components=1 -C /src/isolate \
    && make -C /src/isolate -j2 install BUILD_COMMIT=v2.7 \
    && sed -i -e 's|^cg_root =.*|cg_root = /sys/fs/cgroup/isolate|' \
       -e '/^subid_user =/d' /usr/local/etc/isolate \
    && printf '\nfirst_uid = 60000\nfirst_gid = 60000\nnum_boxes = 1000\n# Go build-cache cleanup needs file locks (clear restriction bit 4).\nsyscall_flags = 65531\n' >> /usr/local/etc/isolate \
    && rm -rf /src/isolate

WORKDIR /src/tester
COPY go.mod go.sum ./
RUN go mod download
COPY api ./api
COPY internal ./internal
COPY cmd ./cmd
RUN go build -o /usr/local/bin/tester ./cmd/tester
COPY scripts/defaults/testlib.h /usr/local/etc/tester/testlib.h
RUN printf 'Docker verification image; host hardware not embedded.\n' > /usr/local/etc/tester/system.txt
COPY docs/behave.toml /usr/local/etc/tester/behave.toml
COPY --chmod=755 scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

WORKDIR /work
COPY docs/behave.toml ./behaviour.toml
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["tester", "verify", "./behaviour.toml"]
