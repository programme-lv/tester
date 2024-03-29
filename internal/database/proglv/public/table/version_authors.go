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

var VersionAuthors = newVersionAuthorsTable("public", "version_authors", "")

type versionAuthorsTable struct {
	postgres.Table

	// Columns
	TaskVersionID postgres.ColumnInteger
	Author        postgres.ColumnString

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type VersionAuthorsTable struct {
	versionAuthorsTable

	EXCLUDED versionAuthorsTable
}

// AS creates new VersionAuthorsTable with assigned alias
func (a VersionAuthorsTable) AS(alias string) *VersionAuthorsTable {
	return newVersionAuthorsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new VersionAuthorsTable with assigned schema name
func (a VersionAuthorsTable) FromSchema(schemaName string) *VersionAuthorsTable {
	return newVersionAuthorsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new VersionAuthorsTable with assigned table prefix
func (a VersionAuthorsTable) WithPrefix(prefix string) *VersionAuthorsTable {
	return newVersionAuthorsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new VersionAuthorsTable with assigned table suffix
func (a VersionAuthorsTable) WithSuffix(suffix string) *VersionAuthorsTable {
	return newVersionAuthorsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newVersionAuthorsTable(schemaName, tableName, alias string) *VersionAuthorsTable {
	return &VersionAuthorsTable{
		versionAuthorsTable: newVersionAuthorsTableImpl(schemaName, tableName, alias),
		EXCLUDED:            newVersionAuthorsTableImpl("", "excluded", ""),
	}
}

func newVersionAuthorsTableImpl(schemaName, tableName, alias string) versionAuthorsTable {
	var (
		TaskVersionIDColumn = postgres.IntegerColumn("task_version_id")
		AuthorColumn        = postgres.StringColumn("author")
		allColumns          = postgres.ColumnList{TaskVersionIDColumn, AuthorColumn}
		mutableColumns      = postgres.ColumnList{}
	)

	return versionAuthorsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		TaskVersionID: TaskVersionIDColumn,
		Author:        AuthorColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
