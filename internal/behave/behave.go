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
	// Either reference a predefined language by id, or provide fields inline.
	LangID        string `toml:"lang_id"`
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
	// Optional registry of languages available for reference via lang_id
	Languages []struct {
		ID            string `toml:"id"`
		LangName      string `toml:"lang_name"`
		CodeFname     string `toml:"code_fname"`
		CompileCmd    string `toml:"compile_cmd"`
		CompiledFname string `toml:"compiled_fname"`
		ExecCmd       string `toml:"exec_cmd"`
	} `toml:"languages"`
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

	// Build language registry if provided
	langByID := make(map[string]SpecLanguage)
	for _, l := range root.Languages {
		if l.ID == "" {
			continue
		}
		langByID[l.ID] = SpecLanguage{
			LangName:      l.LangName,
			CodeFname:     l.CodeFname,
			CompileCmd:    l.CompileCmd,
			CompiledFname: l.CompiledFname,
			ExecCmd:       l.ExecCmd,
		}
	}

	cases := make([]Case, 0, len(root.Suites))
	for _, suite := range root.Suites {
		if len(suite.RequestAOT) == 0 {
			return nil, fmt.Errorf("scenario entry is missing request block")
		}
		reqSpec := suite.RequestAOT[0]

		// Resolve language by id and allow inline overrides
		// 1) Start with base from registry if lang_id is set
		// 2) Overlay inline fields when non-empty
		var eff SpecLanguage
		if reqSpec.Language.LangID != "" {
			base, ok := langByID[reqSpec.Language.LangID]
			if !ok {
				return nil, fmt.Errorf("unknown language id: %s", reqSpec.Language.LangID)
			}
			eff = base
		}
		// Overlay/assign inline values when provided
		if reqSpec.Language.LangName != "" {
			eff.LangName = reqSpec.Language.LangName
		}
		if reqSpec.Language.CodeFname != "" {
			eff.CodeFname = reqSpec.Language.CodeFname
		}
		if reqSpec.Language.CompileCmd != "" {
			eff.CompileCmd = reqSpec.Language.CompileCmd
		}
		if reqSpec.Language.CompiledFname != "" {
			eff.CompiledFname = reqSpec.Language.CompiledFname
		}
		if reqSpec.Language.ExecCmd != "" {
			eff.ExecCmd = reqSpec.Language.ExecCmd
		}

		// Validate required fields after merge
		if eff.LangName == "" || eff.CodeFname == "" || eff.ExecCmd == "" {
			return nil, fmt.Errorf("language specification incomplete; require lang_name, code_fname, exec_cmd (lang_id=%q)", reqSpec.Language.LangID)
		}

		// Map to api.PrLang
		lang := api.PrLang{
			LangName:  eff.LangName,
			CodeFname: eff.CodeFname,
			ExecCmd:   eff.ExecCmd,
		}
		if eff.CompileCmd != "" {
			cc := eff.CompileCmd
			lang.CompileCmd = &cc
		}
		if eff.CompiledFname != "" {
			cf := eff.CompiledFname
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
