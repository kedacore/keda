package scalers

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func GetAwsSqsQueueLength(ctx context.Context, queueURL string) (int32, error) {
	sess, err := session.NewSession(aws.NewConfig().WithRegion("eu-west-1"))
	sqsClient := sqs.New(sess)

	if len(queueURL) == 0 {
		return -1, errors.New("Empty queueUrl is not valid")
	}

	input := &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"All"}),
		QueueUrl:       aws.String(queueURL),
	}

	output, err := sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, nil
	}

	approximateNumberOfMessages, err := strconv.Atoi(*output.Attributes["ApproximateNumberOfMessages"])
	if err != nil {
		return -1, nil
	}

	return int32(approximateNumberOfMessages), nil
}
