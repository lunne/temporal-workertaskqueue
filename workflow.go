package workertaskqueue

import (
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	DefaultTaskQueue = "taskqueue"

	requestSignalName  string = "request"
	defaultTTLduration        = time.Second * 60
)

var activities = &Activities{}

type handler struct {
	logger      log.Logger
	workChannel workflow.Channel
	queue       tasks

	mainCancel workflow.CancelFunc
	cancel     workflow.CancelFunc
}

// TaskQWorkflow...
func TaskQWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	sessionOptions := &workflow.SessionOptions{
		CreationTimeout:  time.Second * 20,
		ExecutionTimeout: defaultTTLduration,
	}
	ctx, err := workflow.CreateSession(ctx, sessionOptions)
	if err != nil {
		return err
	}
	defer workflow.CompleteSession(ctx)

	ctx, mainCancel := workflow.WithCancel(ctx)
	h := &handler{
		workChannel: workflow.NewChannel(ctx),
		queue:       make([]*Task, 0),
		mainCancel:  mainCancel,
		logger:      logger,
	}

	// see that when the workflow is cancelled the timer is also cancelled
	defer func() {
		if h.cancel != nil {
			h.cancel()
		}
	}()

	workflow.GoNamed(ctx, "worker", h.worker)
	workflow.GoNamed(ctx, "sender", h.sender)

	err = h.listen(ctx)
	switch {
	case temporal.IsCanceledError(err):
		ctx, _ = workflow.NewDisconnectedContext(ctx)
	case err != nil:
	}

	if err := h.drainSignals(ctx); err != nil {
		return err
	}

	return err

}

func (h *handler) drainSignals(ctx workflow.Context) error {
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(workflow.GetSignalChannel(ctx, requestSignalName), func(c workflow.ReceiveChannel, more bool) {
		var t Task
		c.ReceiveAsync(&t)
	})

	selector.AddReceive(h.workChannel, func(c workflow.ReceiveChannel, more bool) {
		var t Task
		c.ReceiveAsync(&t)
	})

	for selector.HasPending() {
		selector.Select(ctx)
	}

	return nil
}

func (h *handler) listen(ctx workflow.Context) error {
	selector := workflow.NewSelector(ctx)
	var err error

	selector.AddReceive(workflow.GetSignalChannel(ctx, requestSignalName), func(c workflow.ReceiveChannel, more bool) {
		var t Task
		c.Receive(ctx, &t)

		if h.cancel != nil {
			h.cancel()
		}

		h.queue = append(h.queue, &t)

	}).AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) { // catch the cancel signal
	})

	for ctx.Err() == nil && err == nil {
		selector.Select(ctx)
	}

	return err

}

func (h *handler) sender(ctx workflow.Context) {
	for ctx.Err() == nil {

		// we are blocking on the select in the else if
		if len(h.queue) > 0 {
			var t *Task
			t, h.queue = h.queue.Pop()
			h.workChannel.Send(ctx, t)

		} else {
			var timerCtx workflow.Context
			timerCtx, h.cancel = workflow.WithCancel(ctx)
			selector := workflow.NewSelector(ctx)
			selector.AddFuture(workflow.NewTimer(timerCtx, defaultTTLduration), func(f workflow.Future) {
				if timerCtx.Err() != nil { // timer has been cancelled
					timerCtx = nil
					h.cancel = nil
				} else {
					// ttl has been reached, cancel the workflow
					h.mainCancel()
					return
				}
			})

			selector.Select(ctx)
		}

	}
}

func (h *handler) worker(ctx workflow.Context) {
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) { // catch ctx error
	})

	selector.AddReceive(h.workChannel, func(c workflow.ReceiveChannel, more bool) {
		var task Task
		c.Receive(ctx, &task)

		var data Task
		if err := workflow.ExecuteActivity(optionsWithTimeout(ctx, time.Second*30), activities.ProcessTask, &task).Get(ctx, &data); err != nil {
			h.logger.Warn("could not send process task", "error", err)
		}

		// check if the context was canceled, if it is create a disconneted context and send the response
		if ctx.Err() != nil {
			disconnectContext, cancel := workflow.NewDisconnectedContext(ctx)
			defer cancel()

			if err := h.handleCallback(disconnectContext, &data); err != nil {
				h.logger.Warn("could not send response to request workflow", "error", err)
			}

			return

		} else {
			if err := h.handleCallback(ctx, &data); err != nil {
				h.logger.Warn("could not send response to request workflow", "error", err)
			}
		}

	})

	for ctx.Err() == nil {
		selector.Select(ctx) // wait for the next task
	}

}

func (h *handler) handleCallback(ctx workflow.Context, task *Task) error {
	// send back the response over a signal
	// add the task and payload to the envelope
	_ = workflow.SignalExternalWorkflow(ctx, task.CallbackId, "", task.CallbackSignalName, task)

	return nil

}
