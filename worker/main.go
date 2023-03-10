package main

import (
	"log"

	"go.opentelco.io/workertaskqueue"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// The client and worker are heavyweight objects that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workerOptions := worker.Options{
		EnableSessionWorker: true, // Important for a worker to participate in the session
	}
	w := worker.New(c, workertaskqueue.DefaultTaskQueue, workerOptions)

	w.RegisterWorkflow(workertaskqueue.TaskQWorkflow)
	w.RegisterWorkflow(workertaskqueue.RequestWorkflow)
	w.RegisterActivity(&workertaskqueue.Activities{Client: c})

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
