package utils

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing/models"
)

func CollectProcessRuntimeData(process *isolate.Process) (*models.RuntimeData, error) {
	err := process.Start()
	if err != nil {
		return nil, err
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, err
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, err
	}

	metrics, err := process.Wait()
	if err != nil {
		return nil, err
	}

	return &models.RuntimeData{
		Output: models.RuntimeOutput{
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			ExitCode: metrics.ExitCode,
		},
		Metrics: models.RuntimeMetrics{
			CpuTimeMillis:  int64(metrics.TimeSec * 1000),
			WallTimeMillis: int64(metrics.TimeWallSec * 1000),
			MemoryKBytes:   metrics.CgMemKb,
		},
	}, nil
}

func VerifyContent(fname string, expected []byte) error {
	s, err := storage.GetInstance()
	if err != nil {
		return err
	}

	content, err := s.GetTextFile(fname)
	if err != nil {
		return err
	}
	if string(content) != string(expected) {
		return fmt.Errorf("file %s has content %s, but expected %s", fname, content, expected)
	}

	return nil
}

func VerifySha256(fname string, expected string) error {
	s, err := storage.GetInstance()
	if err != nil {
		return err
	}

	file, err := s.GetTextFile(fname)
	if err != nil {
		return err
	}

	h := sha256.New()
	if _, err := io.Copy(h, bytes.NewReader(file)); err != nil {
		return err
	}

	sha256sum := fmt.Sprintf("%x", h.Sum(nil))
	if sha256sum != expected {
		return fmt.Errorf("file %s has sha256 %s, but expected %s", fname, sha256sum, expected)
	}

	return nil
}
