// Basic usage example for the reghelp-client-go SDK.
//
// Run:
//
//	REGHELP_API_KEY=your_key go run ./examples/basic
//
// Demonstrates:
//   - balance check
//   - getting an iOS push token + waiting for the result
//   - graceful error handling (errors.Is on sentinel errors)
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/REGHELPNET/reghelp-client-go"
)

func main() {
	apiKey := os.Getenv("REGHELP_API_KEY")
	if apiKey == "" {
		log.Fatal("REGHELP_API_KEY is required")
	}

	cli := reghelp.New(apiKey,
		reghelp.WithTimeout(45*time.Second),
		reghelp.WithUserAgent("reghelp-go-example/1.0"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Balance ---------------------------------------------------------
	bal, err := cli.GetBalance(ctx)
	if err != nil {
		fail("get balance", err)
	}
	fmt.Printf("balance: %.4f %s\n", bal.Balance, bal.Currency)

	// 2. iOS push token: create task + poll until done -------------------
	task, err := cli.GetPushToken(ctx, reghelp.PushTokenRequest{
		AppName:   "tgiOS",
		AppDevice: reghelp.AppDeviceIOS,
	})
	if err != nil {
		fail("create push task", err)
	}
	fmt.Printf("task created: id=%s price=%.4f balance=%.4f\n",
		task.ID, task.Price, task.Balance)

	status, err := cli.WaitForResult(ctx, task.ID, reghelp.ServicePush,
		180*time.Second, 2*time.Second)
	if err != nil {
		fail("wait for push", err)
	}
	push := status.(*reghelp.PushStatusResponse)
	fmt.Printf("push token: %s (status=%s)\n", push.Token, push.Status)
}

func fail(stage string, err error) {
	switch {
	case errors.Is(err, reghelp.ErrUnauthorized):
		log.Fatalf("%s: api key rejected", stage)
	case errors.Is(err, reghelp.ErrRateLimit):
		log.Fatalf("%s: rate limited — back off and retry", stage)
	case errors.Is(err, reghelp.ErrTaskNotFound):
		log.Fatalf("%s: task not found (id mismatch?)", stage)
	default:
		log.Fatalf("%s: %v", stage, err)
	}
}
