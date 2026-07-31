// Command example shows how scope rides on a context and stamps its
// attributes onto the logger you read back out of it.
//
// Run it with:
//
//	go run ./scope/.example
//
// A real service configures slog through github.com/psyb0t/slog-configurator;
// this example leans on slog.Default() so it stays dependency-free.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/psyb0t/common-go/scope"
)

const (
	commitSHA     = "deadbeef"
	requestID     = "req-01HXR9"
	userID        = 42
	isRetry       = true
	uploadRatio   = 0.75
	keyCommit     = "commit"
	keyRequestID  = "request_id"
	keyUserID     = "user_id"
	keyRetry      = "retry"
	keyRatio      = "upload_ratio"
	keyAttachment = "attachment"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelDebug},
	)))

	// A build id describes the PROCESS, not any one request, so it goes in the
	// global tier. Everything this binary logs carries it, and — the reason it
	// belongs here rather than in the per-context tier — it is never serialized
	// by ToJSON, so it cannot travel to another service and overwrite that
	// service's own build id.
	scope.SetGlobal(scope.Attr(keyCommit, commitSHA))

	ctx := context.Background()

	// No per-context attributes yet, so only the global tier shows up.
	scope.GetLogger(ctx).Info("before any scope")
	// => msg="before any scope" commit=deadbeef

	handleRequest(ctx)
}

func handleRequest(ctx context.Context) {
	// Set takes as many attributes as you have, and returns a NEW context — a
	// context is immutable, so scope can only ever hand you a derived one.
	// Attr constrains the value to the primitives a scope can render, so bools
	// and floats need no conversion here and a struct would not compile.
	ctx = scope.Set(ctx,
		scope.Attr(keyRequestID, requestID),
		scope.Attr(keyUserID, userID),
		scope.Attr(keyRetry, isRetry),
		scope.Attr(keyRatio, uploadRatio),
	)

	// Every line under this context now carries all four, plus the commit from
	// the global tier.
	scope.GetLogger(ctx).Info("request received")
	// => msg="request received" commit=deadbeef request_id=req-01HXR9
	//    retry=true upload_ratio=0.75 user_id=42

	doWork(ctx)
}

func doWork(ctx context.Context) {
	// A key set twice is replaced, never duplicated — the attributes are
	// applied once, at read time, from a map that holds each key once.
	ctx = scope.Set(ctx, scope.Attr(keyRetry, false))

	// Attributes that should not follow the work further down come back off.
	ctx = scope.Remove(ctx, keyRatio)

	scope.GetLogger(ctx).Debug("working")
	// => msg=working commit=deadbeef request_id=req-01HXR9 retry=false
	//    user_id=42

	// A transient attribute that has no business propagating stays on the
	// logger only — that is what logger.With is still for.
	logger := scope.GetLogger(ctx).With(keyAttachment, "avatar.png")
	logger.Warn("attachment skipped")
	// => msg="attachment skipped" ... attachment=avatar.png
	// ...and it is NOT in scope.Get(ctx) below.

	report(ctx)
}

func report(ctx context.Context) {
	logger := scope.GetLogger(ctx)

	// Get hands back a copy, so this cannot corrupt the context.
	current := scope.Get(ctx)
	current["scratch"] = "local only"
	delete(current, "scratch")

	logger.Info("scope map", "scope", current)
	// => msg="scope map" ... scope=map[request_id:req-01HXR9 retry:false
	//    user_id:42]

	// ToJSON is the wire form: what you would stamp on an outbound HTTP header,
	// gRPC metadata, a queue message, a Temporal header, or a subprocess env.
	// Note what is NOT in it — commit stays behind, because the global tier
	// describes this process and has no business travelling.
	data, err := scope.ToJSON(ctx)
	if err != nil {
		logger.Error("marshal scope", "err", err)

		return
	}

	logger.Info("scope json", "json", string(data))
	// => msg="scope json" ... json={"request_id":"req-01HXR9",
	//    "retry":false,"user_id":42}

	receive(data)
}

// receive is the far side of a hop: a different service, worker, or process.
func receive(data []byte) {
	// Its own build id — deliberately different, to show it survives.
	scope.SetGlobal(scope.Attr(keyCommit, "cafe1234"))

	ctx, err := scope.FromJSON(context.Background(), data)
	if err != nil {
		scope.GetLogger(ctx).Error("unmarshal scope", "err", err)

		return
	}

	scope.GetLogger(ctx).Info("work received")
	// => msg="work received" commit=cafe1234 request_id=req-01HXR9
	//    retry=false user_id=42
	//
	// The work attributes crossed. The build id did not: this side reports
	// cafe1234, the deploy that is actually running this code.
}
