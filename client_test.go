package reghelp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient returns a Client pointed at a freshly minted httptest server.
// The handler is the only thing the test needs to control; the rest uses SDK defaults.
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	defOpts := []Option{WithBaseURL(srv.URL), WithMaxRetries(0), WithRetryDelay(time.Millisecond)}
	defOpts = append(defOpts, opts...)
	return New("test-key", defOpts...), srv
}

func TestHealthCheck(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("want /health, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
	})
	ok, err := cli.HealthCheck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected healthy")
	}
}

func TestGetBalance_ApiKeyForwarded(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("apiKey"); got != "test-key" {
			t.Errorf("apiKey not forwarded, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "balance": 12.34, "currency": "USD",
		})
	})
	bal, err := cli.GetBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bal.Balance != 12.34 || bal.Currency != "USD" {
		t.Errorf("unexpected balance: %#v", bal)
	}
}

func TestGetPushToken_ParamsAndDecode(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for k, want := range map[string]string{
			"appName":   "tgiOS",
			"appDevice": "iOS",
			"appVersion": "12.7",
			"appBuild":   "32933",
			"ref":        "promo",
		} {
			if got := q.Get(k); got != want {
				t.Errorf("param %s = %q, want %q", k, got, want)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "id": "task-1", "service": "push",
			"product": "tg", "price": 0.1, "balance": 99.0,
		})
	})
	res, err := cli.GetPushToken(context.Background(), PushTokenRequest{
		AppName: "tgiOS", AppDevice: AppDeviceIOS,
		AppVersion: "12.7", AppBuild: "32933", Ref: "promo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "task-1" || res.Service != "push" {
		t.Errorf("unexpected response: %#v", res)
	}
}

func TestErrorEnvelope_TaskNotFound(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // server uses 200 + status=error
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error", "id": "TASK_NOT_FOUND",
		})
	})
	_, err := cli.GetPushStatus(context.Background(), "missing-id")
	// /getStatus endpoints use allowErrorStatus=true, so envelope is returned as-is.
	if err != nil {
		t.Fatalf("expected raw envelope, got error: %v", err)
	}
}

func TestErrorEnvelope_OnGetToken_MapsToTyped(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error", "id": "RATE_LIMIT",
		})
	})
	_, err := cli.GetPushToken(context.Background(), PushTokenRequest{AppName: "tg", AppDevice: AppDeviceAndroid})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRateLimit) {
		t.Errorf("expected ErrRateLimit, got %v", err)
	}
}

func TestHTTP401_MapsUnauthorized(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "id": "UNAUTHORIZED"})
	})
	_, err := cli.GetBalance(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRetry_On429ThenSuccess(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(429)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "balance": 1.0, "currency": "USD",
		})
	}, WithMaxRetries(2), WithRetryDelay(time.Millisecond))

	bal, err := cli.GetBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bal.Balance != 1.0 {
		t.Errorf("unexpected balance: %#v", bal)
	}
	if calls != 2 {
		t.Errorf("want 2 calls, got %d", calls)
	}
}

func TestRecaptcha_ProxyApplied(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("proxyType") != "http" || q.Get("proxyAddress") != "p.example.com" ||
			q.Get("proxyPort") != "8080" || q.Get("proxyLogin") != "u" || q.Get("proxyPassword") != "p" {
			t.Errorf("proxy params not applied: %v", q)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "id": "rc-1", "service": "recaptcha",
			"product": "tg", "price": 0.5, "balance": 10.0,
		})
	})
	_, err := cli.GetRecaptchaMobileToken(context.Background(), RecaptchaMobileRequest{
		AppName: "org.telegram.messenger", AppDevice: AppDeviceAndroid,
		AppKey: "key", AppAction: "login",
		Proxy: &ProxyConfig{Type: ProxyTypeHTTP, Address: "p.example.com", Port: 8080, Login: "u", Password: "p"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrity_StdFlowSendsType(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("type"); got != "std" {
			t.Errorf("expected type=std, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "id": "int-1", "service": "integrity",
			"product": "tg", "price": 0.1, "balance": 9.0,
		})
	})
	_, err := cli.GetIntegrityToken(context.Background(), IntegrityTokenRequest{
		AppName: "tg", AppDevice: AppDeviceAndroid,
		Nonce: strings.Repeat("a", 32), AppVersionCode: 12345, TokenType: IntegrityTokenTypeStd,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrity_ClassicFlowOmitsType(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("type"); got != "" {
			t.Errorf("expected no type for classic, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success", "id": "int-1", "service": "integrity",
			"product": "tg", "price": 0.1, "balance": 9.0,
		})
	})
	_, err := cli.GetIntegrityToken(context.Background(), IntegrityTokenRequest{
		AppName: "tg", AppDevice: AppDeviceAndroid,
		Nonce: strings.Repeat("a", 32), AppVersionCode: 12345, // TokenType omitted ⇒ classic
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrity_ValidatesAppVersionCode(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when AppVersionCode is invalid")
	})
	_, err := cli.GetIntegrityToken(context.Background(), IntegrityTokenRequest{
		AppName: "tg", AppDevice: AppDeviceAndroid,
		Nonce: strings.Repeat("a", 32), AppVersionCode: 0,
	})
	if !errors.Is(err, ErrInvalidParameter) {
		t.Errorf("expected ErrInvalidParameter, got %v", err)
	}
}

func TestWaitForResult_PollsUntilDone(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "task-x", "status": "running",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task-x", "status": "done", "token": "real-token",
		})
	})

	st, err := cli.WaitForResult(context.Background(), "task-x", ServicePush,
		2*time.Second, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	push := st.(*PushStatusResponse)
	if push.Token != "real-token" {
		t.Errorf("unexpected status: %#v", push)
	}
	if calls != 3 {
		t.Errorf("want 3 polls, got %d", calls)
	}
}

func TestWaitForResult_Timeout(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task-x", "status": "running",
		})
	})
	_, err := cli.WaitForResult(context.Background(), "task-x", ServicePush,
		20*time.Millisecond, 5*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

func TestSetPushStatus_ErrorWithBalance_StillSuccess(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Server reports a recoverable error envelope with updated balance.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error", "id": "ALREADY_REFUNDED", "balance": 7.5,
		})
	})
	res, err := cli.SetPushStatus(context.Background(), "task", "+15551234567", PushStatusTypeNoSMS)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Error("expected Success=true when balance is present in error envelope")
	}
	if res.Balance == nil || *res.Balance != 7.5 {
		t.Errorf("expected balance 7.5, got %v", res.Balance)
	}
}
