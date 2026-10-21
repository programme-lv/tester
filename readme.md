A code execution worker for https://programme.lv to safely run user submitted code.

Based on great work on untrusted program sandboxing at https://github.com/ioi/isolate .

Tester receives jobs through Core NATS or the legacy AWS SQS listener.
Jobs are specified in JSON format (`api.ExecReq`).
Optional `groups` is a list of scoring units, each a list of 1-based test IDs.
If a test fails (WA/TLE/MLE/RE), later tests whose every group already has a failure are skipped and reported as `test_ignore` instead of being executed.
Omitted or empty `groups` runs every test.
The tester does not assign scores.

The NATS listener can retrieve cache-missing test files through the submitting backend; see [docs/nats-test-files.md](docs/nats-test-files.md).

## Interactive tasks

Interactive requests set `interactor` to a testlib interactor source. Input
and answer files are mandatory for every test; an answer file may be empty,
and the interactor always receives the answer path as its third argument
(`./interactor input.txt output.txt answer.txt`).

The tester runs the solution and the interactor in separate isolate boxes
and connects solution stdout to interactor stdin and the other way around.
Verdicts follow the testlib exit-code convention:

- exit 0: the interactor accepted the solution;
- exit 50–150: partial score `exit - 50` percent (testlib `_pc`, compiled
  with `PC_BASE_EXIT_CODE=50`);
- exit 1 (WA), 2 (PE), 8 (unexpected EOF), or death by SIGPIPE or SIGXFSZ:
  the solution is rejected;
- exit 3 (fail), any other exit code, other signals, and resource-limit
  kills: judge failure, reported as `internal_error` for the whole job.

Acceptance also requires the solution to exit successfully within its
limits. A timeout rejects the test while the solution itself is still
running and is a judge failure once the solution has finished cleanly.
Stderr from the solution and the interactor is diagnostic only and never
affects the verdict.

Limits: the solution runs with `cpu_ms` (+0.5 s CPU extra), `ram_kib`, and
a 20 s wall-clock limit. The interactor has fixed limits: 10 s CPU (+0.5 s),
1 GiB memory, and the same 20 s wall clock.

Prerequisites:
- `isolate` sandbox utility (can run `isolate --cg --init` successfully)
- A NATS server, or AWS credentials when using the SQS listener

For a self-contained Docker verification environment, see [docs/docker.md](docs/docker.md).

To install tester daemon, run `./scripts/install.sh`.
Script will output further instructions to configure and run the service.

TODO: add support for defining programming languages that must be supported in some file
and an entrypoint in the tester executable to test that these languages are supported
on the system.

To run system verification of supported languages and sample program expected results, run
```bash
tester verify ./behaviour.toml
```

On machines without isolate (development hosts, CI), pass `--no-isolate`:

```bash
tester verify ./docs/behave.toml --no-isolate
```

Commands then run directly on the host with best-effort limits: CPU through
`ulimit -t`, wall time by killing the process group, and address space through
`ulimit -v` (raised to at least 256 MiB, since RLIMIT_AS breaks dynamic
loaders where isolate's cgroup limit would not). Memory usage is not reported,
so isolate's cgroup `MLE` verdicts cannot be verified in this mode. It is not
a sandbox: never run untrusted submissions with `--no-isolate`.

When `/usr/local/etc/tester` is not writable, `TESTER_CONFIG_DIR` points at an
alternative directory holding `testlib.h` and `system.txt`.

```bash
tester listen sqs
```

```bash
tester listen nats
```

Alongside, the language compile and run command, we should always send a hello world
or version check command to the tester otherwise it finishes with a signal that process was killed
or something instead of a system error.

We should also check the version of isolate that the system has installed.

isolate installation instructions:
```bash
git clone https://github.com/ioi/isolate.git
cd isolate
make
sudo make install
cd systemd
sudo cp isolate.service /etc/systemd/system/
sudo cp isolate.slice /etc/systemd/system/
sudo systemctl daemon-reexec
sudo systemctl daemon-reload
sudo systemctl enable isolate.service
sudo systemctl start isolate.service
sudo systemctl restart isolate.service
isolate-check-environment
isolate --cg --init
```

You may have to run `sudo systemctl restart isolate.service` after installing.

Giving too little memory will result in a signal 11 for python. 

Deploying tester to server:
```bash
ssh pelekais
```

```bash
cd tester
git pull
systemctl stop tester
./scripts/install.sh
systemctl restart tester
```
