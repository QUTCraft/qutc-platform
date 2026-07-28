package serveradapter

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMockAdapterMakesSimulationExplicit(t *testing.T) {
	adapter := NewMock()
	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Mode != ModeMock || status.State == "online" || status.OnlinePlayers != 0 {
		t.Fatalf("mock status = %+v, want explicit non-online mock state", status)
	}

	result, err := adapter.Execute(context.Background(), "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Accepted || result.Executed || result.Mode != ModeMock {
		t.Fatalf("mock command result = %+v, want accepted but not really executed", result)
	}
}

func TestTimeoutAdapterStopsSlowOperations(t *testing.T) {
	adapter := WithTimeout(blockingAdapter{}, 10*time.Millisecond)
	startedAt := time.Now()
	if _, err := adapter.AddWhitelist(context.Background(), "PlayerOne"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("AddWhitelist() error = %v, want context deadline", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 250*time.Millisecond {
		t.Fatalf("timeout adapter returned after %s", elapsed)
	}
}

type blockingAdapter struct{}

func (blockingAdapter) Name() string { return "blocking-test" }
func (blockingAdapter) Mode() string { return ModeMock }
func (blockingAdapter) Status(ctx context.Context) (Status, error) {
	<-ctx.Done()
	return Status{}, ctx.Err()
}
func (blockingAdapter) Execute(ctx context.Context, _ string) (Result, error) {
	<-ctx.Done()
	return Result{}, ctx.Err()
}
func (blockingAdapter) AddWhitelist(ctx context.Context, _ string) (Result, error) {
	<-ctx.Done()
	return Result{}, ctx.Err()
}

func TestMockAdapterRejectsUntrustedCommandsAndGameIDs(t *testing.T) {
	adapter := NewMock()
	for _, command := range []string{"op somebody", "say ", "list\nstop", "say hello\nop somebody"} {
		if _, err := adapter.Execute(context.Background(), command); !errors.Is(err, ErrCommandNotAllowed) {
			t.Fatalf("Execute(%q) error = %v, want ErrCommandNotAllowed", command, err)
		}
	}
	for _, gameID := range []string{"", "ab", "name with spaces", "player\nop", "name-is-too-long-for-minecraft"} {
		if _, err := adapter.AddWhitelist(context.Background(), gameID); err == nil {
			t.Fatalf("AddWhitelist(%q) unexpectedly succeeded", gameID)
		}
	}
}
