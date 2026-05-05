package commontemporal

import (
	"context"
	"errors"
	"fmt"

	commonerrors "github.com/psyb0t/common-go/errors"
	"github.com/psyb0t/ctxerrors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	TaskTypeWorkflow = "Workflow"
	TaskTypeActivity = "Activity"
)

type Workflow struct {
	client *Client
	ID     string
}

func (c *Client) NewWorkflow(id string) *Workflow {
	return &Workflow{
		client: c,
		ID:     id,
	}
}

func (w *Workflow) GetStatus(
	ctx context.Context,
) (enums.WorkflowExecutionStatus, bool, error) {
	wfExecution, err := w.client.C.DescribeWorkflowExecution(ctx, w.ID, "")
	if err != nil {
		if isWorkflowErrorNotFound(err) {
			return 0, false, commonerrors.ErrNotFound
		}

		return 0, false, ctxerrors.Wrap(err, "failed to get workflow status")
	}

	wfStatus := wfExecution.WorkflowExecutionInfo.Status

	return wfStatus, isWorkflowStatusTerminal(wfStatus), nil
}

func (w *Workflow) GetResult(
	ctx context.Context,
	target any,
) error {
	wfRun := w.client.C.GetWorkflow(ctx, w.ID, "")
	if err := wfRun.Get(ctx, target); err != nil {
		if isWorkflowErrorNotFound(err) {
			return commonerrors.ErrNotFound
		}

		return ctxerrors.Wrap(err, "failed to get workflow result")
	}

	return nil
}

// IsRunning returns true if the workflow is currently in RUNNING state.
// Returns false (no error) if the workflow doesn't exist — useful as a gate
// where "not found" and "not running" are equivalent.
func (w *Workflow) IsRunning(ctx context.Context) (bool, error) {
	status, _, err := w.GetStatus(ctx)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return false, nil
		}

		return false, ctxerrors.Wrap(err, "failed to get workflow status")
	}

	return status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}

// IsWorkflowTypeRunning returns true if any workflow of the given type is
// currently running. Use when the workflow ID is dynamic (random/per-run)
// and only the type is stable. Backed by the visibility store, which is
// eventually consistent — typically <1s lag, so a workflow that just started
// may not be visible immediately.
func (c *Client) IsWorkflowTypeRunning(
	ctx context.Context, workflowType string,
) (bool, error) {
	query := fmt.Sprintf(
		`WorkflowType=%q AND ExecutionStatus=%q`,
		workflowType, "Running",
	)

	resp, err := c.C.ListWorkflow(
		ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Query:    query,
			PageSize: 1,
		},
	)
	if err != nil {
		return false, ctxerrors.Wrapf(
			err, "list workflows by type %q", workflowType,
		)
	}

	return len(resp.GetExecutions()) > 0, nil
}

func (w *Workflow) IsCompletedSuccessfully(ctx context.Context) (bool, error) {
	wfStatus, isTerminal, err := w.GetStatus(ctx)
	if err != nil {
		return false, ctxerrors.Wrap(err, "failed to get workflow status")
	}

	return isTerminal &&
		wfStatus == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, nil
}

func isWorkflowStatusTerminal(status enums.WorkflowExecutionStatus) bool {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		enums.WORKFLOW_EXECUTION_STATUS_FAILED,
		enums.WORKFLOW_EXECUTION_STATUS_CANCELED,
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT,
		enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED:
		return true
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW,
		enums.WORKFLOW_EXECUTION_STATUS_PAUSED:
		return false
	}

	return false
}

func isWorkflowErrorNotFound(err error) bool {
	var notFound *serviceerror.NotFound

	return errors.As(err, &notFound)
}
