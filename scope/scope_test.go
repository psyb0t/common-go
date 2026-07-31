package scope

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLogMessage = "test"

// loggedPairs runs fn against a context whose logger writes JSON to a buffer,
// emits one line, and returns its attributes as ordered key/value pairs.
// Ordered pairs rather than a map because json.Unmarshal into a map silently
// collapses duplicate keys — which is exactly the bug this package must not
// have.
func loggedPairs(
	t *testing.T,
	fn func(ctx context.Context) context.Context,
) [][2]string {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	ctx := GetCtxWithLogger(context.Background(), logger)

	GetLogger(fn(ctx)).Info(testLogMessage)

	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))

	openBrace, err := decoder.Token()
	require.NoError(t, err)
	require.Equal(t, json.Delim('{'), openBrace)

	pairs := [][2]string{}

	for decoder.More() {
		key, err := decoder.Token()
		require.NoError(t, err)

		keyStr, ok := key.(string)
		require.True(t, ok)

		var value any
		require.NoError(t, decoder.Decode(&value))

		if keyStr == slog.TimeKey ||
			keyStr == slog.LevelKey ||
			keyStr == slog.MessageKey {
			continue
		}

		encoded, err := json.Marshal(value)
		require.NoError(t, err)

		pairs = append(pairs, [2]string{keyStr, string(encoded)})
	}

	return pairs
}

func TestGet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		build func(ctx context.Context) context.Context
		want  Scope
	}{
		{
			name: "untouched context",
			build: func(ctx context.Context) context.Context {
				return ctx
			},
			want: Scope{},
		},
		{
			name: "single key",
			build: func(ctx context.Context) context.Context {
				return Set(ctx, Attr("request_id", "abc"))
			},
			want: Scope{"request_id": "abc"},
		},
		{
			name: "mixed primitives",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("user_id", 42))
				ctx = Set(ctx, Attr("retry", true))

				return Set(ctx, Attr("ratio", 0.5))
			},
			want: Scope{
				"user_id": 42,
				"retry":   true,
				"ratio":   0.5,
			},
		},
		{
			name: "re-setting a key keeps the last value",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("stage", "first"))

				return Set(ctx, Attr("stage", "second"))
			},
			want: Scope{"stage": "second"},
		},
		{
			name: "remove drops the key",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("user_id", 42))
				ctx = Set(ctx, Attr("request_id", "abc"))

				return Remove(ctx, "user_id")
			},
			want: Scope{"request_id": "abc"},
		},
		{
			name: "remove of an unset key changes nothing",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("request_id", "abc"))

				return Remove(ctx, "nope")
			},
			want: Scope{"request_id": "abc"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, Get(tc.build(context.Background())))
		})
	}
}

func TestGet_ReturnsACopy(t *testing.T) {
	t.Parallel()

	ctx := Set(context.Background(), Attr("request_id", "abc"))

	mutated := Get(ctx)
	mutated["request_id"] = "tampered"
	mutated["extra"] = "nope"

	assert.Equal(t, Scope{"request_id": "abc"}, Get(ctx))
}

func TestSet_LeavesTheParentContextAlone(t *testing.T) {
	t.Parallel()

	parent := Set(context.Background(), Attr("request_id", "abc"))
	child := Set(parent, Attr("user_id", 42))

	assert.Equal(t, Scope{"request_id": "abc"}, Get(parent))
	assert.Equal(t, Scope{"request_id": "abc", "user_id": 42}, Get(child))
}

func Test_flatten(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   Scope
		want []any
	}{
		{
			name: "nil scope",
			in:   nil,
			want: nil,
		},
		{
			name: "empty scope",
			in:   Scope{},
			want: nil,
		},
		{
			name: "sorted by key",
			in:   Scope{"zeta": 1, "alpha": "a", "mid": true},
			want: []any{"alpha", "a", "mid", true, "zeta", 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, flatten(tc.in))
		})
	}
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		build func(ctx context.Context) context.Context
		want  string
	}{
		{
			name: "empty scope",
			build: func(ctx context.Context) context.Context {
				return ctx
			},
			want: `{}`,
		},
		{
			name: "populated scope",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("request_id", "abc"))

				return Set(ctx, Attr("user_id", 42))
			},
			want: `{"request_id":"abc","user_id":42}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := ToJSON(tc.build(context.Background()))
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(data))
		})
	}
}

// The ctx logger is not incidental output here — keeping it in sync with the
// map is the package's whole contract, so the emitted attrs are the side
// effect worth asserting on.
func TestSet_KeepsTheContextLoggerInSync(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		build func(ctx context.Context) context.Context
		want  [][2]string
	}{
		{
			name: "untouched context logs nothing extra",
			build: func(ctx context.Context) context.Context {
				return ctx
			},
			want: [][2]string{},
		},
		{
			name: "attrs are emitted sorted by key",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("user_id", 42))

				return Set(ctx, Attr("request_id", "abc"))
			},
			want: [][2]string{
				{"request_id", `"abc"`},
				{"user_id", `42`},
			},
		},
		{
			// The bug this guards: stacking logger.With on every Set appends
			// instead of replacing, so re-setting a key emits it twice.
			name: "re-setting a key emits it once",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("stage", "first"))
				ctx = Set(ctx, Attr("stage", "second"))

				return Set(ctx, Attr("stage", "third"))
			},
			want: [][2]string{
				{"stage", `"third"`},
			},
		},
		{
			name: "removed keys stop being emitted",
			build: func(ctx context.Context) context.Context {
				ctx = Set(ctx, Attr("user_id", 42))
				ctx = Set(ctx, Attr("request_id", "abc"))

				return Remove(ctx, "user_id")
			},
			want: [][2]string{
				{"request_id", `"abc"`},
			},
		},
		{
			name: "attrs already on the logger survive scope churn",
			build: func(ctx context.Context) context.Context {
				ctx = GetCtxWithLogger(
					ctx,
					GetLogger(ctx).With("commit", "deadbeef"),
				)
				ctx = Set(ctx, Attr("request_id", "abc"))

				return Remove(ctx, "request_id")
			},
			want: [][2]string{
				{"commit", `"deadbeef"`},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, loggedPairs(t, tc.build))
		})
	}
}

func TestFromJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		build   func(ctx context.Context) context.Context
		data    string
		want    Scope
		wantErr bool
	}{
		{
			name:  "seeds an empty scope",
			build: func(ctx context.Context) context.Context { return ctx },
			data:  `{"request_id":"abc"}`,
			want:  Scope{"request_id": "abc"},
		},
		{
			name:  "empty object leaves the scope alone",
			build: func(ctx context.Context) context.Context { return Set(ctx, Attr("a", "1")) },
			data:  `{}`,
			want:  Scope{"a": "1"},
		},
		{
			name:  "merges with what is already set",
			build: func(ctx context.Context) context.Context { return Set(ctx, Attr("local", "keep")) },
			data:  `{"request_id":"abc"}`,
			want:  Scope{"local": "keep", "request_id": "abc"},
		},
		{
			name:  "incoming wins on a key collision",
			build: func(ctx context.Context) context.Context { return Set(ctx, Attr("request_id", "old")) },
			data:  `{"request_id":"new"}`,
			want:  Scope{"request_id": "new"},
		},
		{
			name:    "garbage is an error and the context is unchanged",
			build:   func(ctx context.Context) context.Context { return Set(ctx, Attr("a", "1")) },
			data:    `not json`,
			want:    Scope{"a": "1"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := FromJSON(tc.build(context.Background()), []byte(tc.data))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.want, Get(got))
		})
	}
}

// The round trip is what crosses a process boundary: ToJSON on the sending
// side, FromJSON on the receiving one. Numbers come back as float64 — JSON has
// one number type — so this pins that down rather than pretending otherwise.
func TestToJSON_FromJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := Set(context.Background(), Attr("request_id", "abc"))
	ctx = Set(ctx, Attr("user_id", 42))
	ctx = Set(ctx, Attr("retry", true))
	ctx = Set(ctx, Attr("ratio", 0.75))

	data, err := ToJSON(ctx)
	require.NoError(t, err)

	got, err := FromJSON(context.Background(), data)
	require.NoError(t, err)

	assert.Equal(t, Scope{
		"request_id": "abc",
		"user_id":    float64(42),
		"retry":      true,
		"ratio":      0.75,
	}, Get(got))

	// The Go values differ by type after the hop, but what actually matters
	// downstream is that the emitted log line does not.
	sent := loggedPairs(t, func(ctx context.Context) context.Context {
		return Set(ctx, Attr("user_id", 42))
	})

	received := loggedPairs(t, func(ctx context.Context) context.Context {
		data, err := ToJSON(Set(context.Background(), Attr("user_id", 42)))
		require.NoError(t, err)

		ctx, err = FromJSON(ctx, data)
		require.NoError(t, err)

		return ctx
	})

	assert.Equal(t, sent, received)
}

// Set and Remove must never touch the base logger. Leaving it untouched is
// what lets Remove work at all — slog has no way to un-apply an attribute
// already baked into a logger — and what stops a re-Set key being emitted
// twice. Everything is applied at read time, by GetLogger.
func TestSet_LeavesTheBaseLoggerUntouched(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	base := slog.New(slog.NewJSONHandler(buf, nil)).With("commit", "deadbeef")

	ctx := GetCtxWithLogger(context.Background(), base)
	ctx = Set(ctx, Attr("request_id", "abc"))
	ctx = Remove(ctx, "request_id")
	ctx = Set(ctx, Attr("user_id", 42))

	// Same pointer after all that churn: nothing rebuilt or replaced it.
	assert.Same(t, base, baseLogger(ctx))

	// And logging through it directly emits only what IT carries.
	baseLogger(ctx).Info(testLogMessage)

	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))

	assert.Equal(t, "deadbeef", record["commit"])
	assert.NotContains(t, record, "user_id")

	// The attribute is on the context, and GetLogger is what applies it.
	assert.Equal(t, Scope{"user_id": 42}, Get(ctx))
}

// A logger is a value: one fetched before a scope change keeps the attributes
// it was built with. This is why the convention is to call GetLogger at the
// point you log rather than holding one in a struct field.
func TestGetLogger_HeldLoggerDoesNotSeeLaterChanges(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	ctx := GetCtxWithLogger(
		context.Background(),
		slog.New(slog.NewJSONHandler(buf, nil)),
	)
	ctx = Set(ctx, Attr("user_id", 42))

	held := GetLogger(ctx)
	ctx = Remove(ctx, "user_id")

	GetLogger(ctx).Info(testLogMessage)
	held.Info(testLogMessage)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)

	var fresh, stale map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &fresh))
	require.NoError(t, json.Unmarshal(lines[1], &stale))

	assert.NotContains(t, fresh, "user_id")
	assert.Equal(t, float64(42), stale["user_id"])
}
