//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

// UseSchema sets a new schema name for all generated table SQL builder types. It is recommended to invoke
// this method only once at the beginning of the program.
func UseSchema(schema string) {
	EvaluationStatuses = EvaluationStatuses.FromSchema(schema)
	EvaluationTestResults = EvaluationTestResults.FromSchema(schema)
	Evaluations = Evaluations.FromSchema(schema)
	FlywaySchemaHistory = FlywaySchemaHistory.FromSchema(schema)
	MarkdownStatements = MarkdownStatements.FromSchema(schema)
	ProblemTags = ProblemTags.FromSchema(schema)
	ProgrammingLanguages = ProgrammingLanguages.FromSchema(schema)
	RuntimeData = RuntimeData.FromSchema(schema)
	RuntimeStatistics = RuntimeStatistics.FromSchema(schema)
	StatementExamples = StatementExamples.FromSchema(schema)
	SubmissionEvaluations = SubmissionEvaluations.FromSchema(schema)
	TaskOrigins = TaskOrigins.FromSchema(schema)
	TaskSubmissions = TaskSubmissions.FromSchema(schema)
	TaskVersionTests = TaskVersionTests.FromSchema(schema)
	TaskVersions = TaskVersions.FromSchema(schema)
	Tasks = Tasks.FromSchema(schema)
	TestingTypes = TestingTypes.FromSchema(schema)
	TestlibCheckers = TestlibCheckers.FromSchema(schema)
	TestlibInteractors = TestlibInteractors.FromSchema(schema)
	TextFiles = TextFiles.FromSchema(schema)
	Users = Users.FromSchema(schema)
	VersionAuthors = VersionAuthors.FromSchema(schema)
}
