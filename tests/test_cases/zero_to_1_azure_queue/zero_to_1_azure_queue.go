package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/kedacore/keda/pkg/scalers"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

func main() {
	args := os.Args[1:]

	if len(args) != 3 {
		fmt.Println("USAGE: [create/get-length] <connectionString> <queueName>")
		os.Exit(1)
	}

	if args[0] == "create" {
		createMessages(args[1], args[2])
	} else if args[0] == "get-length" {
		length, err := scalers.GetAzureQueueLength(context.TODO(), args[1], args[2])
		if err != nil {
			panic(err)
		}
		fmt.Println(length)
	}
}

func createMessages(connectionString, queueName string) {
	ctx, queueURL := getQueueURL(connectionString, queueName)
	messagesURL := queueURL.NewMessagesURL()

	for i := 1; i < 10; i++ {
		_, err := messagesURL.Enqueue(ctx, fmt.Sprintf("This is message %d", i), time.Second*0, time.Minute)
		if err != nil {
			panic(err)
		}
	}
}

func getQueueURL(connectionString, queueName string) (context.Context, azqueue.QueueURL) {
	accountName, accountKey, err := scalers.ParseAzureStorageConnectionString(connectionString)
	if err != nil {
		panic(err)
	}

	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		panic(err)
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))
	serviceURL := azqueue.NewServiceURL(*u, p)
	queueURL := serviceURL.NewQueueURL(queueName)
	ctx := context.TODO()

	_, err = queueURL.Create(ctx, azqueue.Metadata{})
	if err != nil {
		panic(err)
	}
	return ctx, queueURL
}
