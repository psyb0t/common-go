package commontemporal

import (
	"context"
	"errors"
	"time"

	"github.com/psyb0t/ctxerrors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WaitForWorkflowTypeClearActivityName is the registered activity name used
// by WaitForWorkflowTypeClear. Register the activity returned by
// NewWaitForWorkflowTypeClearActivity under this name on the worker that
// runs the calling workflow.
const WaitForWorkflowTypeClearActivityName = "CommonTemporal_WaitForWorkflowTypeClear"

// ErrWorkflowTypeRunning is returned by the wait activity when another
// workflow of the target type is currently running. Temporal's RetryPolicy
// treats this as retryable: each retry is a real wait, durably scheduled by
// the Temporal server (not an in-workflow sleep), so worker restarts don't
// lose the timer.
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

// NewWaitForWorkflowTypeClearActivity returns the activity body that returns
// ErrWorkflowTypeRunning if the target type is still executing, nil if clear.
// Register the result on the worker under WaitForWorkflowTypeClearActivityName:
//
//	w.RegisterActivityWithOptions(
//	    commontemporal.NewWaitForWorkflowTypeClearActivity(client),
//	    activity.RegisterOptions{Name: commontemporal.WaitForWorkflowTypeClearActivityName},
//	)
func NewWaitForWorkflowTypeClearActivity(
	c *Client,
) func(ctx context.Context, workflowType string) error {
	return func(ctx context.Context, workflowType string) error {
		running, err := c.IsWorkflowTypeRunning(ctx, workflowType)
		if err != nil {
			return ctxerrors.Wrap(err, "check workflow type running")
		}

		if running {
			return ErrWorkflowTypeRunning
		}

		return nil
	}
}

// WaitForWorkflowTypeClear blocks the calling workflow until no workflow of
// the given type is running, using Temporal's RetryPolicy as the wait/backoff
// mechanism. Returns nil once clear, or the final error if MaximumAttempts
// is exhausted — caller typically logs and skips its own work in that case.
//
// The activity returned by NewWaitForWorkflowTypeClearActivity must be
// registered on the worker that runs this workflow under the name
// WaitForWorkflowTypeClearActivityName.
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

	if err := workflow.ExecuteActivity(
		actCtx, WaitForWorkflowTypeClearActivityName, workflowType,
	).Get(ctx, nil); err != nil {
		return ctxerrors.Wrapf(
			err, "wait for workflow type %q to clear", workflowType,
		)
	}

	return nil
}
