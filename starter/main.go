package main

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	workertaskqueue "github.com/lunne/temporal-workertaskqueue"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: workertaskqueue.DefaultTaskQueue,
	}

	t := &workertaskqueue.Task{
		Id:       "same-eacht-time",
		Priority: 1,
		Result:   "",
		Created:  time.Now(),
	}

	ctx := context.Background()

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workertaskqueue.RequestWorkflow, t)
	if err != nil {
		log.Fatalln("Unable to execute request workflow", err)
	}

	if err := we.Get(ctx, t); err != nil {
		log.Fatalln("Unable to get result from request workflow", err)
	}

	log.Println("completed request", "id", t.Id, "Result", t.Result)
}
