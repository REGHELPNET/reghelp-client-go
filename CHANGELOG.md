# Changelog

All notable changes to **reghelp-client-go** will be documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-05-20

### Added

- New skill **`attestation`** — WhatsApp Key Attestation cert chains via
  `Client.GetAttestationToken` + `Client.GetAttestationStatus` +
  `Client.PostAttestationFeedback`. Backward-compatible additive change.

  ```go
  task, err := c.GetAttestationToken(ctx, reghelp.AttestationTokenRequest{
      AuthKey: challengeBase64,
  })
  if err != nil { return err }
  result, err := c.WaitForResult(ctx, reghelp.ServiceAttestation, task.ID, 60*time.Second, 2*time.Second)
  if err != nil { return err }
  st := result.(*reghelp.AttestationStatusResponse)
  certChainDERB64 := st.Authorization
  leafKeyB64      := st.LeafPrivateKeyB64
  keyboxID        := st.KeyboxDeviceID

  // Report verdict so the keybox can be canary-quarantined on failure:
  _, _ = c.PostAttestationFeedback(ctx, reghelp.AttestationFeedbackRequest{
      TaskID: task.ID, OK: false, Reason: "downstream rejected",
  })
  ```

- `AttestationStatusResponse`, `AttestationTokenRequest`,
  `AttestationFeedbackRequest`, `ServiceAttestation`.

### Changed

- `Client.do()` is now a thin wrapper around `doMethod()` (internal).
  POST endpoints reuse the same retry/error-mapping pipeline as GET.

## [1.0.0] - 2026-05-20

First public release. Feature parity with Python SDK
[`reghelp_client`](https://github.com/REGHELPNET/reghelp_client) v1.4.0.

### Added
- `Client` with functional options (`WithBaseURL`, `WithHTTPClient`,
  `WithTimeout`, `WithMaxRetries`, `WithRetryDelay`, `WithUserAgent`).
- Push token API (`GetPushToken`, `GetPushStatus`, `SetPushStatus` with refund
  envelope handling for `NOSMS` / `FLOOD` / `BANNED` / `2FA`).
- VoIP push token API (`GetVoIPToken`, `GetVoIPStatus`) — required alongside
  APNS on iOS Telegram registrations.
- Email API (`GetEmail`, `GetEmailStatus`) — iCloud Hide My Email & Gmail OAuth.
- Play Integrity API (`GetIntegrityToken`, `GetIntegrityStatus`) — Classic
  (`MEETS_STRONG_INTEGRITY`) and Standard / Express
  (`MEETS_DEVICE_INTEGRITY`) flows with mandatory `AppVersionCode` validation
  (Key API v2026-05).
- Mobile reCAPTCHA API (`GetRecaptchaMobileToken`, `GetRecaptchaMobileStatus`)
  with optional `ProxyConfig` per request.
- Cloudflare Turnstile API (`GetTurnstileToken`, `GetTurnstileStatus`).
- Balance API (`GetBalance`).
- Health check (`HealthCheck`) — no auth required.
- `WaitForResult` polling helper across all services with timeout and
  cancellation via `context.Context`.
- Typed errors via sentinel values (`ErrUnauthorized`, `ErrRateLimit`,
  `ErrServiceDisabled`, `ErrMaintenance`, `ErrTaskNotFound`,
  `ErrInvalidParameter`, `ErrExternalService`, `ErrTimeout`, `ErrNetwork`,
  `ErrInvalidJSONResult`) usable with `errors.Is` / `errors.As`. Underlying
  `*reghelp.Error` carries HTTP status, server `id`, message, and raw envelope.
- Automatic retry on HTTP 429 and transport errors with exponential back-off
  plus jitter (configurable via `WithMaxRetries` / `WithRetryDelay`).
- Bilingual EN + RU README with API matrix mapping Go ↔ Python method names.
- Comprehensive `httptest`-based unit suite (parameter forwarding, error
  envelope translation, retry-on-429, polling deadlines, Integrity flow
  selection, proxy injection, recoverable refund envelope).
- `examples/basic` end-to-end sample driven by `REGHELP_API_KEY` env var.

### Notes
- Stdlib only — no transitive Go module dependencies.
- Targets Go 1.22+ (uses `errors.Is` and modern `net/http`).
- MIT License preserved from upstream Python SDK.
