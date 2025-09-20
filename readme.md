A code execution worker for https://programme.lv to safely run user submitted code.

Based on great work on untrusted program sandboxing tool at https://github.com/ioi/isolate .

Tester polls an AWS SQS queue for new jobs. Job specification is a JSON object tha

## Features

- Secure code execution using Linux isolate
- Resource constraint enforcement (CPU, memory, time)
- Multiple programming language support
- Test case management and execution
- Real-time result streaming via AWS SQS
- File system isolation and cleanup

## Core Components

### Testing Engine (`/internal/testing`)

The core testing functionality:
- Test case execution and validation
- Output comparison and scoring
- Execution result gathering
- Error handling and reporting
- Support for multiple testing strategies

### Isolation (`/internal/isolate`)

Linux isolate integration:
- Secure sandbox environment
- Resource constraints:
  - CPU time limits
  - Wall clock limits
  - Memory limits
  - Process count limits
- Execution metrics collection
- File system isolation

### File Store (`/internal/filestore`)

File management system:
- Secure file storage and retrieval
- Test case file handling
- Source code file management
- Temporary file cleanup
- Directory structure management

### SQS Gatherer (`/sqsgath`)

Result gathering and communication:
- AWS SQS integration
- Real-time result streaming
- Execution status updates
- Error reporting
- Message formatting and trimming

## Getting Started

### Prerequisites

- Linux operating system
- Go 1.21 or later
- AWS credentials for SQS
- isolate sandbox utility
- Supported compilers/interpreters

### Installation

1. Install system dependencies:
```bash
# Ubuntu/Debian
sudo apt-get install isolate build-essential

# Arch Linux
sudo pacman -S isolate base-devel
```

2. Clone and build:
```bash
git clone https://github.com/programme-lv/tester.git
cd tester
go build
```

3. Configure environment:
```bash
# AWS credentials
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_REGION=eu-central-1

# Tester configuration
export TESTER_QUEUE_URL=your_sqs_queue_url
export TESTER_WORK_DIR=/path/to/work/directory
```

## Usage

### Running the Service

```bash
./go run ./cmd/tester
```

### Configuration

The tester service can be configured through environment variables:

```bash
SUBM_REQ_QUEUE_URL=     # Submission SQS request queue URL
```

### Resource Constraints

Default limits that can be overridden per task:

```go
Constraints {
    CPUTime:     1000,  // ms
    WallTime:    2000,  // ms
    Memory:      256,   // MB
    MaxProc:     1,     // processes
    FileSize:    64,    // MB
}
```

## Development

### Project Structure

```
tester/
├── internal/
│   ├── testing/     # Core testing logic
│   ├── isolate/     # Sandbox integration
│   ├── filestore/   # File management
│   └── config/      # Configuration
├── sqsgath/         # SQS integration
└── cmd/            # CLI commands
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/testing
go test ./internal/filestore
```

### Adding New Languages

1. Implement compiler/interpreter configuration
2. Add resource constraints
3. Configure file extensions
4. Add compilation/execution commands

## Security

- All code runs in isolated sandboxes
- Resource limits strictly enforced
- File system access restricted
- Process creation limited
- Networking disabled in sandbox
- Temporary files cleaned up
