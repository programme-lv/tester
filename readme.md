A code execution worker for https://programme.lv to safely run user submitted code.

Based on great work on untrusted program sandboxing at https://github.com/ioi/isolate .

Tester polls an AWS SQS queue for new jobs. Jobs are specified in JSON format.

Prerequisites:
- `isolate` sandbox utility (can run `isolate --cg --init` successfully)
- AWS credentials for SQS

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

I should define the response format too...

Okay, I came here to implement partial scoring on tasks.


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
systemctl stop tester
cd tester && ./scripts/install.sh
systemctl restart tester
```