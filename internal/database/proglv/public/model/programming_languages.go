//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

type ProgrammingLanguages struct {
	ID               string `sql:"primary_key"`
	FullName         string
	CodeFilename     string
	CompileCmd       *string
	ExecuteCmd       string
	EnvVersionCmd    *string
	HelloWorldCode   *string
	MonacoID         *string
	CompiledFilename *string
}
