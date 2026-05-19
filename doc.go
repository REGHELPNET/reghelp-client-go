// Package reghelp is the official Go SDK for the REGHelp Key API — push tokens
// (APNS, FCM), Cloudflare Turnstile, reCAPTCHA Mobile, Google Play Integrity,
// iCloud Hide My Email, Gmail OAuth and VoIP push, with typed errors,
// configurable retries and zero external dependencies.
//
// It mirrors the Python SDK (github.com/REGHELPNET/reghelp_client) in feature
// coverage and method naming while following Go idioms: context-first APIs,
// typed errors via [errors.As] / [errors.Is], and stdlib-only HTTP transport.
//
// # Quick start
//
//	cli := reghelp.New("your_api_key")
//
//	ctx := context.Background()
//	bal, err := cli.GetBalance(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("balance: %.4f %s", bal.Balance, bal.Currency)
//
//	task, err := cli.GetPushToken(ctx, reghelp.PushTokenRequest{
//	    AppName:   "tgiOS",
//	    AppDevice: reghelp.AppDeviceIOS,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	status, err := cli.WaitForResult(ctx, task.ID, reghelp.ServicePush, 180*time.Second, 2*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("push token: %s", status.(*reghelp.PushStatusResponse).Token)
//
// All services (Push, VoIP, Email, Integrity, Turnstile, RecaptchaMobile) are
// supported. See examples/basic for an end-to-end demo.
package reghelp
