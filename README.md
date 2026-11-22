# REGHelp Go SDK — Push Tokens (APNS, FCM), reCAPTCHA Mobile, Cloudflare Turnstile, Google Play Integrity, iCloud HME & Gmail OAuth

[![Go Reference](https://pkg.go.dev/badge/github.com/REGHELPNET/reghelp-client-go.svg)](https://pkg.go.dev/github.com/REGHELPNET/reghelp-client-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/REGHELPNET/reghelp-client-go)](https://goreportcard.com/report/github.com/REGHELPNET/reghelp-client-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Current release / Текущий релиз: **v1.4.0**.

```bash
go get github.com/REGHELPNET/reghelp-client-go@v1.4.0
```

Official Go client for the **REGHelp Key API**. Automate APNS / FCM push tokens, Google Play Integrity attestation, Cloudflare Turnstile, reCAPTCHA Mobile, iCloud Hide My Email and Gmail OAuth flows from idiomatic Go — context-first, typed errors, zero dependencies (stdlib only), production-tested. Drop-in companion to the Python SDK [`reghelp_client`](https://github.com/REGHELPNET/reghelp_client) for teams that need an `async`/`await`-style Go API for QA automation, mobile attestation, Telegram bot registration and backend integration tests.

> Russian version below / Русская версия ниже.

---

## 📑 Table of contents / Содержание

- [English](#-english)
  - [Features](#-features)
  - [Installation](#-installation)
  - [Quick start](#-quick-start)
  - [API reference](#-api-reference)
  - [Error handling](#-error-handling)
  - [Configuration](#-configuration)
  - [Examples](#-examples)
- [Русская версия](#-русская-версия)
- [FAQ](#-faq)
- [License](#-license)

---

## 🇬🇧 English

### 🚀 Features

- **Push Token API** — APNS, FCM, Telegram iOS / Android / TG-X / TG-Beta push tokens, VoIP push tokens.
- **CAPTCHA API** — Cloudflare Turnstile, reCAPTCHA Mobile, status polling, configurable proxy per request.
- **Device Attestation** — Google Play Integrity Classic (MEETS_STRONG_INTEGRITY) and Standard / Express (MEETS_DEVICE_INTEGRITY) flows.
- **Provider registrar (WhatsApp / Snapchat)** — bind a server-owned Android profile to a client registrar session, then create Play Integrity + Key Attestation on that *same* pinned device: `StartRegistrar`, `GetRegistrarBinding`, `GetProviderAppParams`, `GetBoundIntegrityToken` / `GetBoundIntegrityStatus`, `GetBoundAttestationToken` / `GetBoundAttestationStatus`. Pass `RequestID` for idempotent task creation (sent as `Idempotency-Key`).
- **Email API** — iCloud Hide My Email, Gmail OAuth, verification code polling.
- **Idiomatic Go** — `context.Context` first arg, typed errors via `errors.Is` / `errors.As`, no global state, safe for concurrent use.
- **Zero deps** — only `net/http`, `encoding/json` and friends. Drop into any Go project without dependency surface.
- **Retry & backoff** — automatic retry on HTTP 429 and network errors with exponential backoff + jitter; status polls also survive a temporary HTTP 503 and honour `Retry-After`.
- **Webhook support** — pass a callback URL when creating any task to skip polling.

Built for backend engineers, mobile QA automation teams and Telegram bot developers who need a typed Go SDK for REGHelp services.

### 📦 Installation

```bash
go get github.com/REGHELPNET/reghelp-client-go@latest
```

Requires Go **1.22** or newer (uses `net/http.ServeMux`-style ergonomics, but only stdlib).

### 🔧 Quick start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/REGHELPNET/reghelp-client-go"
)

func main() {
    cli := reghelp.New("your_api_key")
    ctx := context.Background()

    // Check balance.
    bal, err := cli.GetBalance(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("balance: %.4f %s", bal.Balance, bal.Currency)

    // Get a Telegram iOS push token.
    task, err := cli.GetPushToken(ctx, reghelp.PushTokenRequest{
        AppName:   "tgiOS",
        AppDevice: reghelp.AppDeviceIOS,
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("task created: %s", task.ID)

    // Wait for the result.
    status, err := cli.WaitForResult(ctx, task.ID, reghelp.ServicePush,
        180*time.Second, 2*time.Second)
    if err != nil {
        log.Fatal(err)
    }
    push := status.(*reghelp.PushStatusResponse)
    log.Printf("push token: %s", push.Token)
}
```

### 📚 API reference

#### Client construction

```go
cli := reghelp.New("api_key",
    reghelp.WithBaseURL("https://api.reghelp.net"),    // default
    reghelp.WithTimeout(45*time.Second),               // default 30s
    reghelp.WithMaxRetries(5),                         // default 3
    reghelp.WithRetryDelay(2*time.Second),             // default 1s
    reghelp.WithUserAgent("my-bot/1.0"),
    reghelp.WithHTTPClient(custom),                    // optional
)
```

#### 📱 Push tokens (APNS, FCM)

```go
task, _ := cli.GetPushToken(ctx, reghelp.PushTokenRequest{
    AppName:    "tgiOS",                  // tgiOS / tg / tg_beta / tg_x …
    AppDevice:  reghelp.AppDeviceIOS,     // or AppDeviceAndroid
    AppVersion: "12.7",                   // optional
    AppBuild:   "32933",                  // optional
    Ref:        "promo",                  // optional referral tag
})

st, _ := cli.GetPushStatus(ctx, task.ID)
// st.Status == reghelp.TaskStatusDone → st.Token is the push token (hex APNS / FCM string)
```

##### Supported `AppName` values

| App | `AppName` | `AppDevice` |
|---|---|---|
| Telegram iOS Official | `tgiOS` | `AppDeviceIOS` |
| Telegram Android Official (Google Play) | `tg` | `AppDeviceAndroid` |
| Telegram Android Official (Beta) | `tg_beta` | `AppDeviceAndroid` |
| Telegram X (Challegram) | `tg_x` | `AppDeviceAndroid` |

##### Marking a failed push for refund

```go
_, err := cli.SetPushStatus(ctx, task.ID, "+15551234567", reghelp.PushStatusTypeNoSMS)
// PushStatusTypeNoSMS / PushStatusTypeFlood / PushStatusTypeBanned / PushStatusType2FA
```

#### 📞 VoIP push token (iOS — required alongside APNS)

```go
voip, _ := cli.GetVoIPToken(ctx, reghelp.VoIPTokenRequest{AppName: "tgiOS"})
st, _ := cli.GetVoIPStatus(ctx, voip.ID)
```

#### 📧 Email (iCloud HME, Gmail OAuth)

Read inventory before allocating an address:

```go
stock, err := cli.GetEmailStock(ctx, "tg", reghelp.EmailTypeICloud)
if err != nil {
    return err
}
fmt.Println(stock.AppName, stock.Count, stock.UpdatedAt)
```

Application codes: Telegram `tg`, Instagram `ig`, WhatsApp `wa`, or another
configured application. Supported aliases are resolved by the API; `AppName`
is the canonical code. `EmailStockResponse` includes `Service`, `AppName`,
`Count` (`int64`), `UpdatedAt` (`time.Time`, UTC) and `Status`.
This authenticated `GET /email/getStock` call is free and does not create a
task or reserve an address. Stock refreshes about every 30 seconds, with a
10-second API cache; the snapshot does not guarantee subsequent allocation.
Zero is a valid count. Disabled services (currently Gmail) return
`ErrServiceDisabled`; unavailable or stale inventory returns
`ErrExternalService` (`UPSTREAM_ERROR`). Use `errors.Is` to check these errors.

```go
em, _ := cli.GetEmail(ctx, reghelp.EmailRequest{
    AppName:   "tg",
    AppDevice: reghelp.AppDeviceAndroid,
    Phone:     "+15551234567",            // E.164
    Type:      reghelp.EmailTypeICloud,   // or EmailTypeGmail
})

// em.Email is allocated immediately; the verification code arrives later.
res, _ := cli.WaitForResult(ctx, em.ID, reghelp.ServiceEmail, 3*time.Minute, 2*time.Second)
code := res.(*reghelp.EmailStatusResponse).Code
```

#### 🔒 Play Integrity (Classic / Standard)

```go
task, _ := cli.GetIntegrityToken(ctx, reghelp.IntegrityTokenRequest{
    AppName:        "tg",
    AppDevice:      reghelp.AppDeviceAndroid,
    Nonce:          nonceFromTGServer,    // 16..500 URL-safe base64 chars
    AppVersionCode: 31948,                // mandatory since Key API v2026-05
    TokenType:      reghelp.IntegrityTokenTypeClassic, // or IntegrityTokenTypeStd
})
res, _ := cli.WaitForResult(ctx, task.ID, reghelp.ServiceIntegrity, 30*time.Second, 1*time.Second)
token := res.(*reghelp.IntegrityStatusResponse).Token
```

| Flow | Latency | TG flag | Result |
|---|---|---|---|
| Classic (`IntegrityTokenTypeClassic`, default) | ~1-3 s | `MEETS_STRONG_INTEGRITY` | long-lived |
| Standard/Express (`IntegrityTokenTypeStd`) | ~200-600 ms | `MEETS_DEVICE_INTEGRITY` | shorter-lived |

#### 🤖 reCAPTCHA Mobile

```go
task, _ := cli.GetRecaptchaMobileToken(ctx, reghelp.RecaptchaMobileRequest{
    AppName:   "org.telegram.messenger",
    AppDevice: reghelp.AppDeviceAndroid,
    AppKey:    "6Lc-recaptcha-site-key",
    AppAction: "login",
    Proxy: &reghelp.ProxyConfig{
        Type:    reghelp.ProxyTypeHTTP,
        Address: "p.example.com", Port: 8080,
        Login: "user", Password: "pass",
    },
})
res, _ := cli.WaitForResult(ctx, task.ID, reghelp.ServiceRecaptcha, 2*time.Minute, 2*time.Second)
token := res.(*reghelp.RecaptchaMobileStatusResponse).Token
```

#### 🔐 Cloudflare Turnstile

```go
task, _ := cli.GetTurnstileToken(ctx, reghelp.TurnstileRequest{
    URL:     "https://example.com/page",
    SiteKey: "0x4AAAA...",
    Action:  "login",
})
res, _ := cli.WaitForResult(ctx, task.ID, reghelp.ServiceTurnstile, 2*time.Minute, 2*time.Second)
token := res.(*reghelp.TurnstileStatusResponse).Token
```

#### 🔄 Polling

```go
status, err := cli.WaitForResult(ctx, task.ID,
    reghelp.ServicePush,                  // push / email / integrity / recaptcha / turnstile / voip
    180*time.Second,                      // total timeout
    2*time.Second,                        // poll interval
)
```

Returns when the task reaches a terminal state (`done` / `error`), the deadline elapses (`reghelp.ErrTimeout`) or `ctx` is cancelled.

Since **v1.4.0** a temporary HTTP 503 from the status endpoint does not end the
wait. The API answers 503 with `Retry-After` when it cannot read the task store
for a few seconds; the task itself keeps running, so `WaitForResult` keeps
polling until its deadline. A single poll, including its retries, never
outlives that deadline. If the wait times out during a 503 window, the returned
error wraps `ErrTimeout` together with the last status-read error — match it
with `errors.Is(err, reghelp.ErrTimeout)`, not with `==`.

#### 🪝 Webhooks

Pass a `Webhook` URL on any `Get…Token` / `GetEmail` call to receive a POST when the task completes, instead of polling.

### 🚨 Error handling

All errors are wrapped in `*reghelp.Error`. Compare with sentinel errors via `errors.Is`:

```go
_, err := cli.GetPushToken(ctx, req)
switch {
case errors.Is(err, reghelp.ErrUnauthorized):
    // invalid API key
case errors.Is(err, reghelp.ErrRateLimit):
    // back off
case errors.Is(err, reghelp.ErrTaskNotFound):
    // task ID mismatch
case errors.Is(err, reghelp.ErrTimeout):
    // WaitForResult deadline
case errors.Is(err, reghelp.ErrUnavailable):
    // Get*Status only: the status could not be read, the task is not lost.
    // Ask again later; apiErr.RetryAfter holds the server's hint.
case err != nil:
    var apiErr *reghelp.Error
    if errors.As(err, &apiErr) {
        log.Printf("[%d %s] %s", apiErr.Status, apiErr.Code, apiErr.Message)
    }
}
```

Sentinel errors:

| Sentinel | Trigger |
|---|---|
| `ErrUnauthorized` | HTTP 401 / `id=UNAUTHORIZED` |
| `ErrRateLimit` | HTTP 429 / `id=RATE_LIMIT` |
| `ErrServiceDisabled` | `id=SERVICE_DISABLED` |
| `ErrMaintenance` | `id=MAINTENANCE_MODE` |
| `ErrTaskNotFound` | HTTP 404 / `id=TASK_NOT_FOUND` |
| `ErrInvalidParameter` | HTTP 400 / `id=INVALID_PARAM` |
| `ErrExternalService` | HTTP 502 / `id=EXTERNAL_ERROR` |
| `ErrUnavailable` | temporary HTTP 503 on a `Get*Status` call, after all retries |
| `ErrTimeout` | `WaitForResult` deadline reached |
| `ErrNetwork` | transport-level failure |
| `ErrInvalidJSONResult` | non-JSON response body |

#### Temporary HTTP 503 on status polls

`Get*Status` calls repeat the request on a temporary HTTP 503 up to
`WithMaxRetries` times. The pause is the server's `Retry-After` (capped at
30 s); without the header the usual exponential back-off applies, and the pause
stops as soon as `ctx` is done. When the status is still unavailable, the call
returns an `*Error` that matches `ErrUnavailable` and carries `RetryAfter`. It
also matches `ErrExternalService` when the server sends `EXTERNAL_ERROR`, so
check `ErrUnavailable` first if you tell the two apart.

Not repeated on 503: `ErrServiceDisabled`, `ErrMaintenance` and every
task-creating call (`Get…Token`, `GetEmail`). For those a 503 does not prove
the task was not created, and the error is not `ErrUnavailable`.

### ⚙️ Configuration

| Option | Default |
|---|---|
| `WithBaseURL(string)` | `https://api.reghelp.net` |
| `WithHTTPClient(*http.Client)` | `&http.Client{Timeout: 30s}` |
| `WithTimeout(time.Duration)` | `30s` |
| `WithMaxRetries(int)` | `3` |
| `WithRetryDelay(time.Duration)` | `1s` (base for exponential back-off) |
| `WithUserAgent(string)` | `reghelp-client-go/<version>` |

### 🧪 Examples

End-to-end demo: [`examples/basic`](examples/basic/main.go).

```bash
REGHELP_API_KEY=your_key go run ./examples/basic
```

---

## 🇷🇺 Русская версия

Официальный Go SDK для **REGHelp Key API** — push-токены (APNS / FCM), Cloudflare Turnstile, reCAPTCHA Mobile, Google Play Integrity, iCloud Hide My Email и Gmail OAuth. Типизированные ошибки, контекст-первый API, нулевые внешние зависимости (только стандартная библиотека). Подходит для автоматизации регистраций Telegram-аккаунтов на iOS / Android / TG-X / TG-Beta, мобильной аттестации устройств и backend-интеграционных тестов. Пара к Python SDK [`reghelp_client`](https://github.com/REGHELPNET/reghelp_client).

### 🚀 Возможности

- Push-токены: APNS, FCM, Telegram iOS / Android Official / TG-X / TG-Beta, VoIP push (iOS).
- CAPTCHA: Cloudflare Turnstile и reCAPTCHA Mobile с опциональным прокси на каждую задачу.
- Play Integrity: Classic (MEETS_STRONG_INTEGRITY) и Standard / Express (MEETS_DEVICE_INTEGRITY).
- Email: iCloud Hide My Email и Gmail OAuth с автоматическим ожиданием кода.
- Идиоматический Go: `context.Context`-first, типизированные ошибки через `errors.Is` / `errors.As`, безопасность для конкурентного использования.
- Нулевые зависимости: только стандартная библиотека Go (`net/http`, `encoding/json` …).
- Автоматический ретрай на HTTP 429 и сетевых ошибках с экспоненциальной задержкой и джиттером; опрос статуса переживает временный HTTP 503 и учитывает `Retry-After`.
- Поддержка webhook вместо polling — указать `Webhook: "https://…"` в любом `Get…Token` запросе.

### 📦 Установка

```bash
go get github.com/REGHELPNET/reghelp-client-go@latest
```

Требуется Go **1.22+**.

### 🔧 Быстрый старт

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/REGHELPNET/reghelp-client-go"
)

func main() {
    cli := reghelp.New("your_api_key")
    ctx := context.Background()

    bal, _ := cli.GetBalance(ctx)
    log.Printf("Баланс: %.4f %s", bal.Balance, bal.Currency)

    task, _ := cli.GetPushToken(ctx, reghelp.PushTokenRequest{
        AppName:   "tgiOS",
        AppDevice: reghelp.AppDeviceIOS,
    })
    status, _ := cli.WaitForResult(ctx, task.ID, reghelp.ServicePush,
        180*time.Second, 2*time.Second)
    log.Printf("Push токен: %s", status.(*reghelp.PushStatusResponse).Token)
}
```

### 📚 Документация

Полная справка по API — раздел [API reference](#-api-reference) выше (примеры и сигнатуры совпадают с Go-версией). Соответствие методов Python SDK один-в-один:

| Python | Go |
|---|---|
| `RegHelpClient(api_key, ...)` | `reghelp.New(apiKey, opts...)` |
| `await client.get_balance()` | `cli.GetBalance(ctx)` |
| `await client.get_push_token(...)` | `cli.GetPushToken(ctx, req)` |
| `await client.get_voip_token(...)` | `cli.GetVoIPToken(ctx, req)` |
| `await client.get_email(...)` | `cli.GetEmail(ctx, req)` |
| `await client.get_email_stock("tg", EmailType.ICLOUD)` | `cli.GetEmailStock(ctx, "tg", reghelp.EmailTypeICloud)` |
| `await client.get_integrity_token(...)` | `cli.GetIntegrityToken(ctx, req)` |
| `await client.get_recaptcha_mobile_token(...)` | `cli.GetRecaptchaMobileToken(ctx, req)` |
| `await client.get_turnstile_token(...)` | `cli.GetTurnstileToken(ctx, req)` |
| `await client.wait_for_result(id, service, ...)` | `cli.WaitForResult(ctx, id, service, ...)` |

### 🚨 Обработка ошибок

С версии **v1.3.0** `GetEmailStock` возвращает количество доступных адресов
по приложению: `tg` — Telegram, `ig` — Instagram, `wa` — WhatsApp. Поля ответа:
`Service`, `AppName`, `Count`, `UpdatedAt`, `Status`. Запрос бесплатный, без
создания задачи и резервирования адреса. `Count=0` — корректный остаток;
отключённый Gmail возвращает `ErrServiceDisabled`, недоступные или устаревшие
данные — `ErrExternalService`. `UpdatedAt` — время наблюдения в UTC;
обновление примерно раз в 30 секунд, кеш API — 10 секунд. Остаток не гарантирует выдачу.

```go
_, err := cli.GetBalance(ctx)
if errors.Is(err, reghelp.ErrUnauthorized) { /* неверный API-ключ */ }
if errors.Is(err, reghelp.ErrRateLimit)    { /* лимит, ретрай позже */ }
```

С версии **v1.4.0** опрос статуса переживает временный HTTP 503. API отвечает
503 с заголовком `Retry-After`, когда несколько секунд не может прочитать
хранилище задач; сама задача продолжает выполняться. `Get*Status` повторяет
запрос до `WithMaxRetries` раз и ждёт столько, сколько указал сервер (не больше
30 секунд), а `WaitForResult` продолжает опрос до своего срока. Если статус так
и не прочитан, ошибка совпадает с `ErrUnavailable` и несёт `RetryAfter`:

```go
st, err := cli.GetPushStatus(ctx, taskID)
if errors.Is(err, reghelp.ErrUnavailable) {
    // статус временно не прочитать, задача не потеряна — спросить позже
}
```

Не повторяются при 503: отключённый сервис, режим обслуживания и запросы,
создающие задачу (`Get…Token`, `GetEmail`) — для них 503 не доказывает, что
задача не создана. Таймаут `WaitForResult` после 503 оборачивает `ErrTimeout`:
сравнивайте через `errors.Is`, а не `==`.

Полный список sentinel-ошибок — таблица в [English-разделе](#-error-handling).

---

## ❓ FAQ

**Q: Почему ноль зависимостей? Хочется красивый http-клиент.**
A: Передайте свой `*http.Client` через `reghelp.WithHTTPClient(...)`. SDK не лезет в выбор клиента, чтобы не тащить транзитивные зависимости.

**Q: Подходит ли SDK для concurrent-использования?**
A: Да. Один `*reghelp.Client` безопасен для конкурентных вызовов из любого числа горутин — состояние только read-after-construct.

**Q: Где взять `api_key`?**
A: На дашборде [reghelp.net](https://reghelp.net). Тот же ключ, что у Python SDK.

**Q: Чем отличается Classic Integrity от Standard?**
A: Classic (default) делает полный запрос Play Integrity, занимает 1-3 с, токен живёт долго (`MEETS_STRONG_INTEGRITY`). Standard / Express использует кэшированный device token (`MEETS_DEVICE_INTEGRITY`), 200-600 мс, короче живёт.

**Q: Нужно ли вызывать `client.Close()` или `defer`?**
A: Нет — SDK не держит долгоживущих ресурсов сверх `http.Client.Timeout`. Если передан кастомный клиент, его lifecycle на вас.

---

## 📄 License

MIT — see [LICENSE](LICENSE).

Copyright (c) 2026 REGHelp Team.
