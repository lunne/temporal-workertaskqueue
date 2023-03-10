package workertaskqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

type unitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (t *unitTestSuite) SetupTest() {
	t.env = t.NewTestWorkflowEnvironment()
}

func (t *unitTestSuite) AfterTest(suiteName, testName string) {
	t.env.AssertExpectations(t.T())
}

// Run all the tests in the suite
func Test_UnitTestSuite(t *testing.T) {
	suite.Run(t, new(unitTestSuite))
}

func (t *unitTestSuite) Test_Simple() {
	env := t.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(TaskQWorkflow)

	env.SetWorkerOptions(worker.Options{
		EnableSessionWorker:   true, // Important for a worker to participate in the session
		EnableLoggingInReplay: true,
	})
	t1 := &Task{
		Id:                 "task-1",
		Priority:           1,
		CallbackId:         "workflow-id",
		CallbackSignalName: requestSignalName,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(requestSignalName, t1)

	}, 0)

	t2 := &Task{
		Id:                 "task-2",
		Priority:           1,
		CallbackId:         "workflow-id",
		CallbackSignalName: requestSignalName,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(requestSignalName, t2)

	}, 10*time.Second)

	env.OnActivity(activities.ProcessTask, mock.Anything, mock.Anything).Return(t1, nil).Times(1)
	env.OnSignalExternalWorkflow("default-test-namespace", "workflow-id", "", requestSignalName, mock.MatchedBy(func(v *Task) bool { return true })).Return(nil).Times(1)

	env.OnActivity(activities.ProcessTask, mock.Anything, mock.Anything).Return(t2, nil).Times(1)
	env.OnSignalExternalWorkflow("default-test-namespace", "workflow-id", "", requestSignalName, mock.MatchedBy(func(v *Task) bool { return true })).Return(nil).Times(1)

	env.ExecuteWorkflow(TaskQWorkflow)

	t.True(env.IsWorkflowCompleted())
	t.NoError(env.GetWorkflowError())

	err := env.GetWorkflowError()
	if err != nil {
		fmt.Println(err)
	}
	env.AssertExpectations(t.T())

}
