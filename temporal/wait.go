package commontemporal

import (
	"context"
	"errors"
	"time"

	"github.com/psyb0t/ctxerrors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ErrWorkflowTypeRunning is returned by WaitForWorkflowTypeClear's activity
// when another workflow of the target type is currently running. Temporal's
// RetryPolicy treats this as retryable: each retry is a real wait, durably
// scheduled by the Temporal server (not an in-workflow sleep), so worker
// restarts don't lose the timer.
var ErrWorkflowTypeRunning = errors.New("workflow of given type is still running")

// WaitForWorkflowTypeClearConfig shapes the wait strategy. Each "retry" is
// the Temporal-scheduled gap between activity attempts. Defaults give:
// 5min, 10min, 15min, 15min — total ~45min wait across 4 attempts before
// giving up.
type WaitForWorkflowTypeClearConfig struct {
	StartToCloseTimeout time.Duration
	InitialInterval     time.Duration
	BackoffCoefficient  float64
	MaximumInterval     time.Duration
	MaximumAttempts     int32
}

func (c *WaitForWorkflowTypeClearConfig) applyDefaults() {
	if c.StartToCloseTimeout == 0 {
		c.StartToCloseTimeout = 30 * time.Second
	}

	if c.InitialInterval == 0 {
		c.InitialInterval = 5 * time.Minute
	}

	if c.BackoffCoefficient == 0 {
		c.BackoffCoefficient = 2.0
	}

	if c.MaximumInterval == 0 {
		c.MaximumInterval = 15 * time.Minute
	}

	if c.MaximumAttempts == 0 {
		c.MaximumAttempts = 4
	}
}

// WaitForWorkflowTypeClearActivity is an activity body that returns
// ErrWorkflowTypeRunning if the target type is still executing, nil if
// clear. Register the *Client on your worker so this method is exposed
// as an activity, then call WaitForWorkflowTypeClear from your workflow.
func (c *Client) WaitForWorkflowTypeClearActivity(
	ctx context.Context, workflowType string,
) error {
	running, err := c.IsWorkflowTypeRunning(ctx, workflowType)
	if err != nil {
		return ctxerrors.Wrap(err, "check workflow type running")
	}

	if running {
		return ErrWorkflowTypeRunning
	}

	return nil
}

// WaitForWorkflowTypeClear blocks the calling workflow until no workflow of
// the given type is running, using Temporal's RetryPolicy as the wait/backoff
// mechanism. Returns nil once clear, or the final error if MaximumAttempts
// is exhausted — caller typically logs and skips its own work in that case.
//
// The *Client must be registered as an activity provider on the worker that
// runs this workflow (worker.RegisterActivity(client)) so the activity body
// is reachable.
func WaitForWorkflowTypeClear(
	ctx workflow.Context,
	workflowType string,
	cfg WaitForWorkflowTypeClearConfig,
) error {
	cfg.applyDefaults()

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: cfg.StartToCloseTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    cfg.InitialInterval,
			BackoffCoefficient: cfg.BackoffCoefficient,
			MaximumInterval:    cfg.MaximumInterval,
			MaximumAttempts:    cfg.MaximumAttempts,
		},
	})

	var c *Client

	if err := workflow.ExecuteActivity(
		actCtx, c.WaitForWorkflowTypeClearActivity, workflowType,
	).Get(ctx, nil); err != nil {
		return ctxerrors.Wrapf(
			err, "wait for workflow type %q to clear", workflowType,
		)
	}

	return nil
}
