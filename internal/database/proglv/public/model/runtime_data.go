//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

type RuntimeData struct {
	ID              int64 `sql:"primary_key"`
	Stdout          *string
	Stderr          *string
	TimeMillis      *int64
	MemoryKibibytes *int64
	TimeWallMillis  *int64
	ExitCode        *int64
}
