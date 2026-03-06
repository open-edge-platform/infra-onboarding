// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/tink/internal/proto"
)

// mockContainerManager is a mock implementation of ContainerManager for testing.
type mockContainerManager struct {
	pullImageFunc func(ctx context.Context, image string) error
}

func (m *mockContainerManager) CreateContainer(_ context.Context, _ []string, _ string, _ *proto.WorkflowAction, _, _ bool) (string, error) {
	return "", nil
}

func (m *mockContainerManager) StartContainer(_ context.Context, _ string) error {
	return nil
}

func (m *mockContainerManager) WaitForContainer(_ context.Context, _ string) (proto.State, error) {
	return proto.State_STATE_SUCCESS, nil
}

func (m *mockContainerManager) WaitForFailedContainer(_ context.Context, _ string, _ chan proto.State) {
}

func (m *mockContainerManager) RemoveContainer(_ context.Context, _ string) error {
	return nil
}

func (m *mockContainerManager) PullImage(ctx context.Context, image string) error {
	if m.pullImageFunc != nil {
		return m.pullImageFunc(ctx, image)
	}
	return nil
}

func TestPullImageWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	var callCount int32
	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				atomic.AddInt32(&callCount, 1)
				return nil
			},
		},
		retries:       3,
		retryInterval: 10 * time.Millisecond,
	}

	err := w.pullImageWithRetry(context.Background(), logr.Discard(), "test-image:latest")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("expected PullImage to be called 1 time, got %d", got)
	}
}

func TestPullImageWithRetry_SuccessAfterRetries(t *testing.T) {
	var callCount int32
	failTimes := int32(2) // fail the first 2 attempts, succeed on the 3rd

	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				n := atomic.AddInt32(&callCount, 1)
				if n <= failTimes {
					return fmt.Errorf("transient error attempt %d", n)
				}
				return nil
			},
		},
		retries:       3,
		retryInterval: 10 * time.Millisecond,
	}

	err := w.pullImageWithRetry(context.Background(), logr.Discard(), "test-image:latest")
	if err != nil {
		t.Fatalf("expected no error after retries, got: %v", err)
	}
	if got := atomic.LoadInt32(&callCount); got != 3 {
		t.Fatalf("expected PullImage to be called 3 times, got %d", got)
	}
}

func TestPullImageWithRetry_AllAttemptsFail(t *testing.T) {
	var callCount int32
	maxRetries := 3

	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				atomic.AddInt32(&callCount, 1)
				return fmt.Errorf("persistent error")
			},
		},
		retries:       maxRetries,
		retryInterval: 10 * time.Millisecond,
	}

	err := w.pullImageWithRetry(context.Background(), logr.Discard(), "test-image:latest")
	if err == nil {
		t.Fatal("expected error after all retries exhausted, got nil")
	}

	expectedCalls := int32(maxRetries + 1) // 1 initial + 3 retries
	if got := atomic.LoadInt32(&callCount); got != expectedCalls {
		t.Fatalf("expected PullImage to be called %d times, got %d", expectedCalls, got)
	}

	expectedMsg := fmt.Sprintf("failed to pull image after %d attempts", maxRetries+1)
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Fatalf("expected error message to start with %q, got: %v", expectedMsg, err)
	}
}

func TestPullImageWithRetry_ContextCancelled(t *testing.T) {
	var callCount int32

	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				atomic.AddInt32(&callCount, 1)
				return fmt.Errorf("transient error")
			},
		},
		retries:       5,
		retryInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay so the first attempt fails but the retry backoff gets cancelled
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := w.pullImageWithRetry(ctx, logr.Discard(), "test-image:latest")
	if err == nil {
		t.Fatal("expected error when context is cancelled, got nil")
	}

	// Should have made at least 1 call (the initial attempt) but not all 6
	got := atomic.LoadInt32(&callCount)
	if got < 1 {
		t.Fatalf("expected at least 1 call, got %d", got)
	}
	if got >= 6 {
		t.Fatalf("expected fewer than 6 calls due to cancellation, got %d", got)
	}
}

func TestPullImageWithRetry_ExponentialBackoff(t *testing.T) {
	var timestamps []time.Time
	var callCount int32
	failTimes := int32(3)

	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				atomic.AddInt32(&callCount, 1)
				timestamps = append(timestamps, time.Now())
				n := atomic.LoadInt32(&callCount)
				if n <= failTimes {
					return fmt.Errorf("transient error")
				}
				return nil
			},
		},
		retries:       4,
		retryInterval: 50 * time.Millisecond,
	}

	err := w.pullImageWithRetry(context.Background(), logr.Discard(), "test-image:latest")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(timestamps) < 4 {
		t.Fatalf("expected at least 4 timestamps, got %d", len(timestamps))
	}

	// Verify exponential backoff: each gap should be roughly double the previous.
	// Gap 0->1: ~50ms, Gap 1->2: ~100ms, Gap 2->3: ~200ms
	// We use generous tolerances since CI can be slow.
	expectedMinBackoffs := []time.Duration{
		30 * time.Millisecond,  // 1st retry: ~50ms
		60 * time.Millisecond,  // 2nd retry: ~100ms
		120 * time.Millisecond, // 3rd retry: ~200ms
	}

	for i := 0; i < len(timestamps)-1 && i < len(expectedMinBackoffs); i++ {
		gap := timestamps[i+1].Sub(timestamps[i])
		if gap < expectedMinBackoffs[i] {
			t.Errorf("gap between attempt %d and %d was %v, expected at least %v",
				i+1, i+2, gap, expectedMinBackoffs[i])
		}
	}
}

func TestPullImageWithRetry_ZeroRetries(t *testing.T) {
	var callCount int32

	w := &Worker{
		containerManager: &mockContainerManager{
			pullImageFunc: func(_ context.Context, _ string) error {
				atomic.AddInt32(&callCount, 1)
				return fmt.Errorf("error")
			},
		},
		retries:       0,
		retryInterval: 10 * time.Millisecond,
	}

	err := w.pullImageWithRetry(context.Background(), logr.Discard(), "test-image:latest")
	if err == nil {
		t.Fatal("expected error with zero retries, got nil")
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("expected PullImage to be called 1 time with zero retries, got %d", got)
	}
}
