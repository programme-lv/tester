package natsfile

import (
	"time"

	"github.com/nats-io/nats.go"
)

type subscription interface {
	NextMsg(time.Duration) (*nats.Msg, error)
	Unsubscribe() error
}

type broker interface {
	newInbox() string
	subscribeSync(string) (subscription, error)
	publishRequest(string, string, []byte) error
	flush() error
}

type natsBroker struct {
	nc *nats.Conn
}

func (b natsBroker) newInbox() string {
	return nats.NewInbox()
}

func (b natsBroker) subscribeSync(subject string) (subscription, error) {
	return b.nc.SubscribeSync(subject)
}

func (b natsBroker) publishRequest(subject, reply string, data []byte) error {
	return b.nc.PublishRequest(subject, reply, data)
}

func (b natsBroker) flush() error {
	return b.nc.Flush()
}
