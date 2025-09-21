package sqsgath

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func NewSqsResponseQueueGatherer(evalUuid string, responseSqsUrl string) *sqsResQueueGatherer {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	return &sqsResQueueGatherer{
		sqsClient: sqs.NewFromConfig(cfg),
		queueUrl:  responseSqsUrl,
		evalUuid:  evalUuid,
	}
}
