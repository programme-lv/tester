package statuses

type Status string

const (
	Received              Status = "R"
	Compiling             Status = "C"
	Testing               Status = "T"
	Finished              Status = "F"
	CompilationError      Status = "CE"
	Accepted              Status = "AC"
	WrongAnswer           Status = "WA"
	TimeLimitExceeded     Status = "TLE"
	MemoryLimitExceeded   Status = "MLE"
	IdlenessLimitExceeded Status = "ILE"
	Ignored               Status = "IG"
	RuntimeError          Status = "RE"
	InternalServerError   Status = "ISE"
)

/*
	InQueue               Status = "IQ"
	Rejected              Status = "RJ"
	PresentationError     Status = "PE"
	PartiallyCorrect      Status = "PT"
	SecurityViolation     Status = "SV"
*/
