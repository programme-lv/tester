package sqsgath

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func (s *sqsResQueueGatherer) send(msg interface{}) {
	b, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("failed to marshal message: %v", err)
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueUrl),
		MessageBody: aws.String(string(b)),
	})

	if err != nil {
		log.Fatalf("failed to send message: %v", err)
	}
}
