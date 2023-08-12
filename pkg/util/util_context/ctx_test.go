package util_context

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWithTimeoutAndReset(t *testing.T) {
	ctx, cancel, reset := WithTimeoutAndReset(context.Background(), 1*time.Second)

	// Check that the context is not done
	fmt.Printf("Checking that the context is not done after 0.5 s \n")
	time.Sleep(500 * time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatalf("context should not be done")
	default: // have a default case to avoid blocking
	}

	fmt.Printf("Resetting every 500 ms and checking that the context is not done (5 times)\n")
	for i := 0; i < 5; i++ {
		fmt.Printf(" attempt %d\n", i)
		reset()
		time.Sleep(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Fatalf("context should not be done")
		default:
		}
	}

	fmt.Printf("Checking that the context is done after 1.5 s\n")
	time.Sleep(1500 * time.Millisecond)
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("context should be done")
	}

	defer cancel()
}

func TestWithTimeoutAndResetCancel(t *testing.T) {
	ctx, cancel, _ := WithTimeoutAndReset(context.Background(), 1*time.Second)

	// Check that the context is not done
	fmt.Printf("Checking that the context is not done after 0.5 s \n")
	time.Sleep(500 * time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatalf("context should not be done")
	default: // have a default case to avoid blocking
	}

	fmt.Printf("Cancelling context and checking that the context is done\n")
	cancel()

	select {
	case <-ctx.Done():
	default:
		t.Fatalf("context should be done")
	}

	defer cancel()
}
