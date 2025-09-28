package behave

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	"github.com/programme-lv/tester/api"
)

// SpecTest is a single test case in the behaviour file
type SpecTest struct {
	In  string `toml:"in"`
	Ans string `toml:"ans"`
}

// SpecLanguage describes language commands in the behaviour file
type SpecLanguage struct {
	LangName      string `toml:"lang_name"`
	CodeFname     string `toml:"code_fname"`
	CompileCmd    string `toml:"compile_cmd"`
	CompiledFname string `toml:"compiled_fname"`
	ExecCmd       string `toml:"exec_cmd"`
}

// SpecRequest represents a request block inside a scenario entry
type SpecRequest struct {
	Code     string       `toml:"code"`
	Tests    []SpecTest   `toml:"tests"`
	Language SpecLanguage `toml:"language"`
	Limits   SpecLimits   `toml:"limits"`
}

// SpecLimits describes resource limits for a scenario request
type SpecLimits struct {
	CpuMs  int32 `toml:"cpu_ms"`
	WallMs int32 `toml:"wall_ms"`
	RamKiB int32 `toml:"ram_kib"`
}

// SpecTestVerdict represents an expected verdict for a test result
type SpecTestVerdict struct {
	Verdict string `toml:"verdict"`
}

// SpecExpect describes expected overall status and per-test verdicts
type SpecExpect struct {
	Status      string            `toml:"status"`
	TestResults []SpecTestVerdict `toml:"test_results"`
}

// specSuite maps to [[scenarios]] entries. The request is written as an array-of-table in the example,
// so we model it as a slice and use the first element.
type specSuite struct {
	Description string        `toml:"description"`
	RequestAOT  []SpecRequest `toml:"request"`
	Expect      SpecExpect    `toml:"expect"`
}

type specRoot struct {
	Suites []specSuite `toml:"scenarios"`
}

// Case is a runnable scenario converted from TOML
type Case struct {
	Name    string
	Request api.ExecReq
	Expect  SpecExpect
}

// Parse reads a behaviour TOML file and converts it to runnable cases using api.ExecReq
func Parse(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read behaviour file: %w", err)
	}
	var root specRoot
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	cases := make([]Case, 0, len(root.Suites))
	for _, suite := range root.Suites {
		if len(suite.RequestAOT) == 0 {
			return nil, fmt.Errorf("scenario entry is missing request block")
		}
		reqSpec := suite.RequestAOT[0]

		// Map language
		lang := api.PrLang{
			LangName:  reqSpec.Language.LangName,
			CodeFname: reqSpec.Language.CodeFname,
			ExecCmd:   reqSpec.Language.ExecCmd,
		}
		if reqSpec.Language.CompileCmd != "" {
			cc := reqSpec.Language.CompileCmd
			lang.CompileCmd = &cc
		}
		if reqSpec.Language.CompiledFname != "" {
			cf := reqSpec.Language.CompiledFname
			lang.CompiledFname = &cf
		}

		// Build tests
		apiTests := make([]api.Test, 0, len(reqSpec.Tests))
		for _, t := range reqSpec.Tests {
			in := t.In
			ans := t.Ans
			apiTests = append(apiTests, api.Test{
				In:  api.File{Content: &in},
				Ans: api.File{Content: &ans},
			})
		}

		// Apply limits with sensible defaults if not provided
		cpuMs := reqSpec.Limits.CpuMs
		if cpuMs == 0 {
			cpuMs = 2000
		}
		ramKiB := reqSpec.Limits.RamKiB
		if ramKiB == 0 {
			ramKiB = 256 * 1024
		}

		execReq := api.ExecReq{
			Uuid:   uuid.NewString(),
			Code:   reqSpec.Code,
			Lang:   lang,
			Tests:  apiTests,
			CpuMs:  cpuMs,
			RamKiB: ramKiB,
		}

		cases = append(cases, Case{
			Name:    suite.Description,
			Request: execReq,
			Expect:  suite.Expect,
		})
	}

	return cases, nil
}
