//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

type EvaluationTestResults struct {
	ID             int64 `sql:"primary_key"`
	EvaluationID   int64
	EvalStatusID   string
	TaskVTestID    int64
	ExecRDataID    *int64
	CheckerRDataID *int64
}
