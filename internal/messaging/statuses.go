package messaging

type Status string

const (
	InQueue               Status = "IQ"
	Compiling             Status = "C"
	Testing               Status = "T"
	Finished              Status = "F"
	CompilationError      Status = "CE"
	Rejected              Status = "RJ"
	Accepted              Status = "AC"
	PartiallyCorrect      Status = "PT"
	WrongAnswer           Status = "WA"
	PresentationError     Status = "PE"
	TimeLimitExceeded     Status = "TLE"
	MemoryLimitExceeded   Status = "MLE"
	IdlenessLimitExceeded Status = "ILE"
	Ignored               Status = "IG"
	RuntimeError          Status = "RE"
	SecurityViolation     Status = "SV"
	InternalServerError   Status = "ISE"
)
