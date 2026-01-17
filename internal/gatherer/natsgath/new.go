package natsgath

import (
	"github.com/nats-io/nats.go"
)

// New creates a new NATS gatherer that streams responses to the given inbox subject.
func New(nc *nats.Conn, evalUuid string, inbox string) *natsGatherer {
	return &natsGatherer{
		nc:       nc,
		inbox:    inbox,
		evalUuid: evalUuid,
	}
}
