package commontemporal

import (
	"github.com/psyb0t/ctxerrors"
	"go.temporal.io/sdk/worker"
)

type Worker struct {
	client *Client
	W      worker.Worker
}

func (c *Client) NewWorker(
	taskQueue string,
	opts worker.Options,
) *Worker {
	return &Worker{
		client: c,
		W: worker.New(
			c.C,
			taskQueue,
			opts,
		),
	}
}

func (w *Worker) RegisterWorkflow(workflow any) {
	w.W.RegisterWorkflow(workflow)
}

func (w *Worker) RegisterWorkflows(workflows ...any) {
	for _, workflow := range workflows {
		w.RegisterWorkflow(workflow)
	}
}

func (w *Worker) RegisterActivity(activity any) {
	w.W.RegisterActivity(activity)
}

func (w *Worker) RegisterActivities(activities ...any) {
	for _, activity := range activities {
		w.RegisterActivity(activity)
	}
}

func (w *Worker) Run() error {
	if err := w.W.Run(worker.InterruptCh()); err != nil {
		return ctxerrors.Wrap(err, "failed to run worker")
	}

	return nil
}

func (w *Worker) Stop() {
	w.W.Stop()
}
