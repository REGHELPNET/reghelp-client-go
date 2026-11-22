package reghelp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

func writeUnavailable(w http.ResponseWriter, retryAfter string) {
	if retryAfter != "" {
		w.Header().Set("Retry-After", retryAfter)
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]any{"detail": "EXTERNAL_ERROR"})
}

func TestGetStatus_Retries503ThenSucceeds(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/push/getStatus" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if calls < 3 {
			writeUnavailable(w, "")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t1", "status": "done", "token": "tok"})
	}, WithMaxRetries(3))

	st, err := cli.GetPushStatus(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if st.Token != "tok" || calls != 3 {
		t.Errorf("token=%q calls=%d", st.Token, calls)
	}
}

func TestGetStatus_503Exhausted_ReturnsUnavailable(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeUnavailable(w, "0")
	}, WithMaxRetries(2))

	_, err := cli.GetPushStatus(context.Background(), "t1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	// Callers matching the broader sentinel keep working.
	if !errors.Is(err, ErrExternalService) {
		t.Errorf("expected ErrExternalService too, got %v", err)
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 503 || apiErr.Code != "EXTERNAL_ERROR" {
		t.Errorf("unexpected error shape: %#v", apiErr)
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestGetStatus_RetriesPlainText503FromProxy(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("no available server"))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t1", "status": "pending"})
	}, WithMaxRetries(1))

	if _, err := cli.GetTurnstileStatus(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("want 2 calls, got %d", calls)
	}
}

func TestGetStatus_PlainText503Exhausted_IsUnavailableNotJSONError(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("no available server"))
	})
	_, err := cli.GetPushStatus(context.Background(), "t1")
	if !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrInvalidJSONResult) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
}

func TestGetStatus_Deliberate503NotRetried(t *testing.T) {
	for code, sentinel := range map[string]error{
		"SERVICE_DISABLED": ErrServiceDisabled,
		"MAINTENANCE_MODE": ErrMaintenance,
	} {
		calls := 0
		cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"detail": code})
		}, WithMaxRetries(3))

		_, err := cli.GetPushStatus(context.Background(), "t1")
		if !errors.Is(err, sentinel) {
			t.Errorf("%s: expected %v, got %v", code, sentinel, err)
		}
		if errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: must not be ErrUnavailable", code)
		}
		if calls != 1 {
			t.Errorf("%s: want 1 call, got %d", code, calls)
		}
	}
}

func TestPaidRequest_NotRetriedOn503(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeUnavailable(w, "0")
	}, WithMaxRetries(3))

	_, err := cli.GetTurnstileToken(context.Background(), TurnstileRequest{
		URL: "https://example.com", SiteKey: "0x4AAAAAAA",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	// A 503 on a task-creating request is not marked as safe to repeat.
	if errors.Is(err, ErrUnavailable) || !errors.Is(err, ErrExternalService) {
		t.Errorf("expected ErrExternalService only, got %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call, got %d", calls)
	}
}

func TestRetryAfterParsing(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
		ok   bool
	}{
		{"2", 2 * time.Second, true},
		{" 0 ", 0, true},
		{"9999", maxRetryAfter, true},
		{"", 0, false},
		{"-1", 0, false},
		{"soon", 0, false},
		{"Wed, 21 Oct 2026 07:28:00 GMT", 0, false},
	}
	for _, tc := range cases {
		got, ok := retryAfter(tc.raw)
		if got != tc.want || ok != tc.ok {
			t.Errorf("retryAfter(%q) = %v, %v; want %v, %v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestGetStatus_503RetryStopsOnContextCancel(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeUnavailable(w, "30")
	}, WithMaxRetries(3))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := cli.GetPushStatus(ctx, "t1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context deadline, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("retry pause ignored ctx: %v", elapsed)
	}
}

func TestWaitForResult_SurvivesLong503Window(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 5 {
			writeUnavailable(w, "0")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t1", "status": "done", "token": "tok"})
	}) // maxRetries=0: every 503 reaches the wait loop

	st, err := cli.WaitForResult(context.Background(), "t1", ServicePush, 5*time.Second, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if st.(*PushStatusResponse).Token != "tok" || calls != 6 {
		t.Errorf("calls=%d status=%#v", calls, st)
	}
}

func TestWaitForResult_TimesOutWhen503NeverEnds(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeUnavailable(w, "0")
	})
	_, err := cli.WaitForResult(context.Background(), "t1", ServicePush, 40*time.Millisecond, 5*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

func TestWaitForResult_OtherErrorsStillEndTheWait(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "TASK_NOT_FOUND"})
	})
	_, err := cli.WaitForResult(context.Background(), "t1", ServicePush, time.Second, time.Millisecond)
	if !errors.Is(err, ErrTaskNotFound) || calls != 1 {
		t.Errorf("expected ErrTaskNotFound after 1 call, got %v after %d", err, calls)
	}
}

func TestWaitForResult_DeadlineBoundsRequestLevelRetries(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeUnavailable(w, "2")
	}, WithMaxRetries(3)) // the request layer alone would sleep 3 x Retry-After

	start := time.Now()
	_, err := cli.WaitForResult(context.Background(), "t1", ServicePush, 200*time.Millisecond, 50*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("wait overshot its timeout: %v", elapsed)
	}
	if calls != 1 {
		t.Errorf("want 1 call, got %d", calls)
	}
}

func TestWaitForResult_TimeoutAfterGoodPollHasNoStale503(t *testing.T) {
	calls := 0
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeUnavailable(w, "0")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t1", "status": "pending"})
	})
	_, err := cli.WaitForResult(context.Background(), "t1", ServicePush, 60*time.Millisecond, 5*time.Millisecond)
	if err != ErrTimeout {
		t.Errorf("expected the bare ErrTimeout, got %v", err)
	}
}

func TestWaitForResult_CallerContextCancelWins(t *testing.T) {
	cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeUnavailable(w, "30")
	}, WithMaxRetries(3))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := cli.WaitForResult(ctx, "t1", ServicePush, 10*time.Second, 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrTimeout) {
		t.Errorf("expected the caller's ctx error, got %v", err)
	}
}
