package api

// ExecReq stands for code execution request
// Execution runs code on an ordered set of tests
// It's not within scope to assign a score here
// Server that runs code is called "tester"
// The name "tester" sounds better than executioner
type ExecReq struct {
	EvalUuid string `json:"eval_uuid"`

	Code string `json:"code"`
	Lang PrLang `json:"language"`

	Tests []Test `json:"tests"`

	Checker    *string `json:"checker"`
	Interactor *string `json:"interactor"`

	// Using integers is easier to work with than floats
	CpuMillis int32 `json:"cpu_millis"`
	// Kibibytes are more precise than kilobytes
	MemoryKiB int32 `json:"memory_kib"`
}

// Test or test case is a pair of input and answer
// If checker or interactor is present, answer may stay unused
type Test struct {
	ID  int32 `json:"id"` // = idx+1 in test slice
	In  File  `json:"in"`
	Ans File  `json:"ans"`
}

type File struct {
	// SHA to check if file exists in cache
	Sha256 *string `json:"sha256"`
	// URL to download file if missing
	Url *string `json:"url"`
	// Content directly as an alternative to URL
	Content *string `json:"content"`
}

// Defines programming language compilation, execution commands
type PrLang struct {
	// Practically only for logging purposes
	LangName string `json:"lang_name"`

	// Place code in sandbox as this file
	CodeFname string `json:"code_fname"`

	// If programming lang has compilation step
	CompileCmd *string `json:"compile_cmd"`

	// After compilation, extract executable from sandbox
	CompiledFname *string `json:"compiled_fname"`

	// With executable in sandbox, run this command
	ExecCmd string `json:"exec_cmd"`
}
