package messaging

type ResponseCorrelation struct {
	UnixMillis  int64 `json:"unix_millis"`
	RandomInt63 int64 `json:"random_int_63"`
}
