package natsgath

import (
	"encoding/json"
	"log"
)

func (s *natsGatherer) send(msg interface{}) {
	b, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}

	if err := s.nc.Publish(s.inbox, b); err != nil {
		log.Printf("failed to publish message to NATS: %v", err)
	}
}
