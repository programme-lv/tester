package isolate

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestDirectModeCommand(t *testing.T) {
	SetDirectMode(true)
	defer SetDirectMode(false)

	box, err := NewBox()
	if err != nil {
		t.Fatal(err)
	}
	defer box.Close()

	if err := box.AddFile("hello.sh", []byte("echo hello")); err != nil {
		t.Fatal(err)
	}
	cmd, err := box.Command("bash hello.sh", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(cmd.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := cmd.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if metrics.ExitCode != 0 {
		t.Fatalf("exit code %d", metrics.ExitCode)
	}
	if !strings.Contains(string(out), "hello") {
		t.Fatalf("unexpected output %q", out)
	}
}

func TestDirectModeWallTimeout(t *testing.T) {
	SetDirectMode(true)
	defer SetDirectMode(false)

	box, err := NewBox()
	if err != nil {
		t.Fatal(err)
	}
	defer box.Close()

	cmd, err := box.Command("sleep 5", &Constraints{WallTimeLimInSec: 0.2})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	metrics, err := cmd.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("wall timeout not enforced, took %v", elapsed)
	}
	if metrics.Status == nil || *metrics.Status != "TO" {
		t.Fatalf("expected TO status, got %v", metrics.Status)
	}
}
