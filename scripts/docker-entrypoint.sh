#!/bin/bash
set -euo pipefail

cg=/sys/fs/cgroup
if [[ ! -f "$cg/cgroup.controllers" ]]; then
    echo 'tester requires cgroup v2.' >&2
    exit 1
fi
if [[ ! -w "$cg/cgroup.procs" ]]; then
    echo 'tester requires writable cgroups; run with --privileged --cgroupns=private.' >&2
    exit 1
fi

# The cgroup mount must be rooted at this container (private cgroup namespace).
# Move the entrypoint and any init process into a leaf before enabling controllers:
# cgroup v2 does not allow processes in an internal domain node.
mkdir -p "$cg/runtime" "$cg/isolate"
mapfile -t pids < "$cg/cgroup.procs"
for pid in "${pids[@]}"; do
    printf '%s\n' "$pid" > "$cg/runtime/cgroup.procs"
done
printf '+cpuset +memory\n' > "$cg/cgroup.subtree_control"
printf '+cpuset +memory\n' > "$cg/isolate/cgroup.subtree_control"

exec "$@"
