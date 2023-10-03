package isolate

import (
	"fmt"
)

type Constraints struct {
	CpuTimeLimInSec      float64
	ExtraCpuTimeLimInSec float64
	WallTimeLimInSec     float64
	MemoryLimitInKB      int
	MaxProcesses         int
	MaxOpenFiles         int
}

func DefaultConstraints() Constraints {
	return Constraints{
		CpuTimeLimInSec:      50.0,
		ExtraCpuTimeLimInSec: 0.5,
		WallTimeLimInSec:     10.0,
		MemoryLimitInKB:      2048000,
		MaxProcesses:         128,
		MaxOpenFiles:         128,
	}
}

func (constraints *Constraints) ToArgs() []string {
	return []string{
		constraints.MemLimArg(),
		constraints.CpuTimeLimArg(),
		constraints.ExtraCpuTimeLimArg(),
		constraints.WallTimeLimArg(),
		constraints.MaxProcessesArg(),
		constraints.MaxOpenFilesArg(),
	}
}

func (constraints *Constraints) MemLimArg() string {
	return fmt.Sprintf("--mem=%d", constraints.MemoryLimitInKB)
}

func (constraints *Constraints) CpuTimeLimArg() string {
	return fmt.Sprintf("--time=%f", constraints.CpuTimeLimInSec)
}

func (constraints *Constraints) ExtraCpuTimeLimArg() string {
	return fmt.Sprintf("--extra-time=%f", constraints.ExtraCpuTimeLimInSec)
}

func (constraints *Constraints) WallTimeLimArg() string {
	return fmt.Sprintf("--wall-time=%f", constraints.WallTimeLimInSec)
}

func (constraints *Constraints) MaxProcessesArg() string {
	return fmt.Sprintf("--processes=%d", constraints.MaxProcesses)
}

func (constraints *Constraints) MaxOpenFilesArg() string {
	return fmt.Sprintf("--open-files=%d", constraints.MaxOpenFiles)
}
