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