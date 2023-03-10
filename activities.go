package workertaskqueue

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var optionsWithTimeout = func(ctx workflow.Context, t time.Duration) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:              DefaultTaskQueue,
		StartToCloseTimeout:    t,
		ScheduleToCloseTimeout: t,
		HeartbeatTimeout:       t,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Millisecond * 200,
			MaximumAttempts:        1,
			NonRetryableErrorTypes: []string{},
		},
	})

}

var withDefaultActivityOptions = func(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           DefaultTaskQueue,
		StartToCloseTimeout: time.Second * 60,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second * 5,
			BackoffCoefficient:     2.0,
			MaximumInterval:        time.Minute,
			NonRetryableErrorTypes: []string{},
		},
	})

}

type Activities struct {
	Client client.Client
}

func (a *Activities) ProcessTask(ctx context.Context, task *Task) (*Task, error) {
	task.Result = "task processed"
	return task, nil
}

func (a *Activities) SendRequest(ctx context.Context, task *Task) error {
	info := activity.GetInfo(ctx)

	task.CallbackId = info.WorkflowExecution.ID
	task.CallbackSignalName = callbackSignalChName

	workflowId := fmt.Sprintf("taskqueue-%s-%d", task.Id, task.Priority)
	_, err := a.Client.SignalWithStartWorkflow(ctx,
		workflowId,
		requestSignalName,
		task,
		client.StartWorkflowOptions{
			TaskQueue: DefaultTaskQueue,
			ID:        workflowId,
		},
		TaskQWorkflow,
	)

	return err
}
