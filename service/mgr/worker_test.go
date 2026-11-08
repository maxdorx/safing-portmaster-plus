package mgr

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var (
	errDeliberateFailure = errors.New("deliberate failure")
	errFirstRunFailure   = errors.New("first-run failure")
)

func TestWorkerInfo(t *testing.T) { //nolint:paralleltest
	mgr := New("test")
	mgr.Go("test func one", testFunc1)
	mgr.Go("test func two", testFunc2)
	mgr.Go("test func three", testFunc3)
	defer mgr.Cancel()

	time.Sleep(100 * time.Millisecond)

	info, err := mgr.WorkerInfo(nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Waiting != 3 {
		t.Errorf("expected three waiting workers")
	}

	fmt.Printf("%+v\n", info)
}

func testFunc1(ctx *WorkerCtx) error {
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
	}
	return nil
}

func testFunc2(ctx *WorkerCtx) error {
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
	}
	return nil
}

func testFunc3(ctx *WorkerCtx) error {
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
	}
	return nil
}

func TestRunWorkerDoesNotMutateCallerContext(t *testing.T) { //nolint:paralleltest
	m := New("test")
	defer m.Cancel()

	base := &WorkerCtx{
		ctx:    m.Ctx(),
		logger: m.logger.With("worker", "test"),
	}
	wantErr := errDeliberateFailure

	for _, tc := range []struct {
		name string
		fn   func(*WorkerCtx) error
	}{
		{"error only", func(_ *WorkerCtx) error { return wantErr }},
		{"self-cancel then error", func(w *WorkerCtx) error { w.Cancel(); return wantErr }},
	} {
		_, err := m.runWorker(base, tc.fn)
		if !errors.Is(err, wantErr) {
			t.Fatalf("%s: expected %v, got %v", tc.name, wantErr, err)
		}
		if base.Ctx().Err() != nil {
			t.Fatalf("%s: caller context must not be canceled after run, got: %v", tc.name, base.Ctx().Err())
		}
	}
}

// TestManagerGoRetryReceivesLiveContext is the regression test for the fix:
// workers using select-on-Done must not silently exit on retry.
func TestManagerGoRetryReceivesLiveContext(t *testing.T) { //nolint:paralleltest
	m := New("test")
	defer m.Cancel()

	var runs atomic.Int32
	done := make(chan struct{})

	m.Go("retry-test", func(w *WorkerCtx) error {
		switch runs.Add(1) {
		case 1:
			return errFirstRunFailure
		case 2:
			select {
			case <-w.Done():
				t.Error("retry context was already canceled")
			default:
			}
			close(done)
		}
		return nil
	})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out after %d runs", runs.Load())
	}
	if runs.Load() != 2 {
		t.Errorf("expected 2 runs, got %d", runs.Load())
	}
}
