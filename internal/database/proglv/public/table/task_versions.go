//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/postgres"
)

var TaskVersions = newTaskVersionsTable("public", "task_versions", "")

type taskVersionsTable struct {
	postgres.Table

	// Columns
	ID            postgres.ColumnInteger
	TaskID        postgres.ColumnInteger
	ShortCode     postgres.ColumnString
	FullName      postgres.ColumnString
	TimeLimMs     postgres.ColumnInteger
	MemLimKb      postgres.ColumnInteger
	TestingTypeID postgres.ColumnString
	Origin        postgres.ColumnString
	CreatedAt     postgres.ColumnTimestampz
	UpdatedAt     postgres.ColumnTimestampz
	CheckerID     postgres.ColumnInteger
	InteractorID  postgres.ColumnInteger

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type TaskVersionsTable struct {
	taskVersionsTable

	EXCLUDED taskVersionsTable
}

// AS creates new TaskVersionsTable with assigned alias
func (a TaskVersionsTable) AS(alias string) *TaskVersionsTable {
	return newTaskVersionsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new TaskVersionsTable with assigned schema name
func (a TaskVersionsTable) FromSchema(schemaName string) *TaskVersionsTable {
	return newTaskVersionsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new TaskVersionsTable with assigned table prefix
func (a TaskVersionsTable) WithPrefix(prefix string) *TaskVersionsTable {
	return newTaskVersionsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new TaskVersionsTable with assigned table suffix
func (a TaskVersionsTable) WithSuffix(suffix string) *TaskVersionsTable {
	return newTaskVersionsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newTaskVersionsTable(schemaName, tableName, alias string) *TaskVersionsTable {
	return &TaskVersionsTable{
		taskVersionsTable: newTaskVersionsTableImpl(schemaName, tableName, alias),
		EXCLUDED:          newTaskVersionsTableImpl("", "excluded", ""),
	}
}

func newTaskVersionsTableImpl(schemaName, tableName, alias string) taskVersionsTable {
	var (
		IDColumn            = postgres.IntegerColumn("id")
		TaskIDColumn        = postgres.IntegerColumn("task_id")
		ShortCodeColumn     = postgres.StringColumn("short_code")
		FullNameColumn      = postgres.StringColumn("full_name")
		TimeLimMsColumn     = postgres.IntegerColumn("time_lim_ms")
		MemLimKbColumn      = postgres.IntegerColumn("mem_lim_kb")
		TestingTypeIDColumn = postgres.StringColumn("testing_type_id")
		OriginColumn        = postgres.StringColumn("origin")
		CreatedAtColumn     = postgres.TimestampzColumn("created_at")
		UpdatedAtColumn     = postgres.TimestampzColumn("updated_at")
		CheckerIDColumn     = postgres.IntegerColumn("checker_id")
		InteractorIDColumn  = postgres.IntegerColumn("interactor_id")
		allColumns          = postgres.ColumnList{IDColumn, TaskIDColumn, ShortCodeColumn, FullNameColumn, TimeLimMsColumn, MemLimKbColumn, TestingTypeIDColumn, OriginColumn, CreatedAtColumn, UpdatedAtColumn, CheckerIDColumn, InteractorIDColumn}
		mutableColumns      = postgres.ColumnList{TaskIDColumn, ShortCodeColumn, FullNameColumn, TimeLimMsColumn, MemLimKbColumn, TestingTypeIDColumn, OriginColumn, CreatedAtColumn, UpdatedAtColumn, CheckerIDColumn, InteractorIDColumn}
	)

	return taskVersionsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:            IDColumn,
		TaskID:        TaskIDColumn,
		ShortCode:     ShortCodeColumn,
		FullName:      FullNameColumn,
		TimeLimMs:     TimeLimMsColumn,
		MemLimKb:      MemLimKbColumn,
		TestingTypeID: TestingTypeIDColumn,
		Origin:        OriginColumn,
		CreatedAt:     CreatedAtColumn,
		UpdatedAt:     UpdatedAtColumn,
		CheckerID:     CheckerIDColumn,
		InteractorID:  InteractorIDColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
