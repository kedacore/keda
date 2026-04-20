package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func noopWorkflow(ctx workflow.Context) error {
	return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
}

func noopActivity(ctx context.Context) error {
	return nil
}

func main() {
	addr := flag.String("address", "localhost:7233", "Temporal server address")
	taskQueue := flag.String("task-queue", "omes-test", "Task queue name")
	deploymentName := flag.String("deployment-name", "", "Worker deployment name")
	buildID := flag.String("build-id", "", "Build ID")
	flag.Parse()

	c, err := client.DialContext(context.Background(), client.Options{
		HostPort: *addr,
	})
	if err != nil {
		log.Fatal("unable to create client:", err)
	}
	defer c.Close()

	opts := worker.Options{}
	if *deploymentName != "" && *buildID != "" {
		opts.DeploymentOptions = worker.DeploymentOptions{
			UseVersioning: true,
			Version: worker.WorkerDeploymentVersion{
				DeploymentName: *deploymentName,
				BuildID:        *buildID,
			},
		}
	}

	w := worker.New(c, *taskQueue, opts)
	w.RegisterWorkflowWithOptions(noopWorkflow, workflow.RegisterOptions{
		Name:               "workflow_with_single_noop_activity",
		VersioningBehavior: workflow.VersioningBehaviorAutoUpgrade,
	})
	w.RegisterActivityWithOptions(noopActivity, activity.RegisterOptions{Name: "noop_activity"})

	if err := w.Start(); err != nil {
		log.Fatal("unable to start worker:", err)
	}

	log.Printf("worker started on %s (deployment=%s, build=%s)", *taskQueue, *deploymentName, *buildID)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	w.Stop()
}
