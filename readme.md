A code execution worker for https://programme.lv to safely run user submitted code.

Based on great work on untrusted program sandboxing at https://github.com/ioi/isolate .

Tester receives jobs through Core NATS or the legacy AWS SQS listener.
Jobs are specified in JSON format (`api.ExecReq`).
Optional `groups` is a list of scoring units, each a list of 1-based test IDs.
If a test fails (WA/TLE/MLE/RE), later tests whose every group already has a failure are skipped and reported as `test_ignore` instead of being executed.
Omitted or empty `groups` runs every test.
The tester does not assign scores.

The NATS listener can retrieve cache-missing test files through the submitting backend; see [docs/nats-test-files.md](docs/nats-test-files.md).

Prerequisites:
- `isolate` sandbox utility (can run `isolate --cg --init` successfully)
- A NATS server, or AWS credentials when using the SQS listener

To install tester daemon, run `./scripts/install.sh`.
Script will output further instructions to configure and run the service.

TODO: add support for defining programming languages that must be supported in some file
and an entrypoint in the tester executable to test that these languages are supported
on the system.

To run system verification of supported languages and sample program expected results, run
```bash
tester verify ./behaviour.toml
```

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