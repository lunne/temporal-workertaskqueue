package workertaskqueue

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

const (
	callbackSignalChName  = "callback"
	defaultRequestTimeout = time.Second * 30
)

func RequestWorkflow(ctx workflow.Context, task *Task) (*Task, error) {

	if err := workflow.ExecuteActivity(withDefaultActivityOptions(ctx), activities.SendRequest, task).Get(ctx, nil); err != nil {
		return nil, err
	}

	callbackChan := workflow.GetSignalChannel(ctx, callbackSignalChName)
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timer := workflow.NewTimer(timerCtx, defaultRequestTimeout)

	var mainErr error

	// setup the selector and the TTL, wait for the session workflows to send a signal with the result
	selector := workflow.NewSelector(timerCtx).AddReceive(callbackChan, func(channel workflow.ReceiveChannel, more bool) {
		var task *Task
		channel.Receive(ctx, &task)
		cancelTimer() // cancel timeout so it does not hit and cancel at this point

	}).AddFuture(timer, func(f workflow.Future) {
		if err := f.Get(timerCtx, nil); err != nil {
			// timer was cancelled
		} else {
		}

		mainErr = fmt.Errorf("timeout reached")

	})
	selector.Select(ctx)

	if mainErr != nil {
		return nil, fmt.Errorf("could not complete request: %w", mainErr)
	}

	return task, nil
}
