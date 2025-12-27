package main

import (
	"fmt"
	"log"

	"github.com/KanavDutta/SignalFence-v1/pkg/signalfence"
)

func main() {
	// Create a rate limiter with 10 tokens, refilling at 2 tokens/second
	limiter, err := signalfence.NewRateLimiter(
		signalfence.WithDefaults(10, 2.0),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("SignalFence Simple Example")
	fmt.Println("==========================")
	fmt.Println("Limits: 10 tokens, refills at 2 tokens/second")
	fmt.Println()

	// Simulate 15 requests from a user
	for i := 1; i <= 15; i++ {
		decision, err := limiter.Allow("user-123")
		if err != nil {
			log.Fatal(err)
		}

		if decision.Allowed {
			fmt.Printf("Request %2d: ✓ ALLOWED  (Remaining: %d/%d)\n",
				i, decision.Remaining, decision.Limit)
		} else {
			fmt.Printf("Request %2d: ✗ DENIED   (Retry after: %.1f seconds)\n",
				i, decision.RetryAfter.Seconds())
		}
	}

	fmt.Println()
	fmt.Println("First 10 requests were allowed (used all tokens).")
	fmt.Println("Remaining 5 requests were denied (bucket empty).")
	fmt.Println()
	fmt.Println("Try running the demo server:")
	fmt.Println("  go run cmd/demo/main.go")
}
