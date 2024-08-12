package utils

import (
	"fmt"
	"io"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
)

func CollectProcessRuntimeData(process *isolate.Process) (*internal.RuntimeData, error) {
	err := process.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	metrics, err := process.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for process: %w", err)
	}

	return &internal.RuntimeData{
		Stdout:          stringPtr(string(stdout)),
		Stderr:          stringPtr(string(stderr)),
		ExitCode:        metrics.ExitCode,
		CpuTimeMillis:   int64(metrics.TimeSec * 1000),
		WallTimeMillis:  int64(metrics.TimeWallSec * 1000),
		MemoryKibiBytes: metrics.CgMemKb,
	}, nil
}

func stringPtr(s string) *string {
	return &s
}

// func VerifyContent(fname string, expected []byte) error {
// 	s, err := storage.GetInstance()
// 	if err != nil {
// 		return err
// 	}

// 	content, err := s.GetTextFile(fname)
// 	if err != nil {
// 		return err
// 	}
// 	if string(content) != string(expected) {
// 		return fmt.Errorf("file %s has content %s, but expected %s", fname, content, expected)
// 	}

// 	return nil
// }

// func VerifySha256(fname string, expected string) error {
// 	s, err := storage.GetInstance()
// 	if err != nil {
// 		return err
// 	}

// 	file, err := s.GetTextFile(fname)
// 	if err != nil {
// 		return err
// 	}

// 	h := sha256.New()
// 	if _, err := io.Copy(h, bytes.NewReader(file)); err != nil {
// 		return err
// 	}

// 	sha256sum := fmt.Sprintf("%x", h.Sum(nil))
// 	if sha256sum != expected {
// 		return fmt.Errorf("file %s has sha256 %s, but expected %s", fname, sha256sum, expected)
// 	}

// 	return nil
// }
