package main

import (
	"context"
	"flag"
	"log"

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

	log.Printf("worker starting on %s (deployment=%s, build=%s)", *taskQueue, *deploymentName, *buildID)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal("worker exited with error:", err)
	}
}
