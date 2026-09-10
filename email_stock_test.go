package reghelp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestGetEmailStock(t *testing.T) {
	for _, app := range []string{"tg", "ig", "wa", "telegram"} {
		for _, count := range []int64{0, 37493} {
			t.Run(app+"/"+strconv.FormatInt(count, 10), func(t *testing.T) {
				canonical := app
				if app == "telegram" {
					canonical = "tg"
				}
				cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					if r.Method != "GET" || r.URL.Path != "/email/getStock" || len(q) != 3 || q.Get("appName") != app || q.Get("type") != "icloud" || q.Get("apiKey") != "test-key" {
						t.Error("incorrect stock request")
					}
					if r.Header.Get("Idempotency-Key") != "" {
						t.Error("stock must not create a paid task")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{
						"status": "success", "service": "icloud", "appName": canonical,
						"count": count, "updatedAt": "2026-09-10T14:51:26.989000Z",
					})
				})
				stock, err := cli.GetEmailStock(context.Background(), app, EmailTypeICloud)
				if err != nil {
					t.Fatal(err)
				}
				if stock.Count != count || stock.AppName != canonical || stock.Service != EmailTypeICloud || !stock.UpdatedAt.Equal(time.Date(2026, 9, 10, 14, 51, 26, 989000000, time.UTC)) {
					t.Fatalf("unexpected stock: %+v", stock)
				}
			})
		}
	}
}

func TestGetEmailStockErrors(t *testing.T) {
	for code, want := range map[string]error{
		"SERVICE_DISABLED": ErrServiceDisabled, "UPSTREAM_ERROR": ErrExternalService,
		"EXTERNAL_ERROR": ErrExternalService, "UNAUTHORIZED": ErrUnauthorized,
	} {
		t.Run(code, func(t *testing.T) {
			cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("type") != "gmail" {
					t.Error("wrong email type")
				}
				w.WriteHeader(503)
				_ = json.NewEncoder(w).Encode(map[string]string{"detail": code})
			})
			stock, err := cli.GetEmailStock(context.Background(), "tg", EmailTypeGmail)
			if stock != nil || !errors.Is(err, want) {
				t.Fatalf("unexpected stock/error: %+v / %v", stock, err)
			}
		})
	}
}

func TestGetEmailStockInvalidCount(t *testing.T) {
	for _, count := range []any{nil, -1, 1.5, true, "10"} {
		cli, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success", "service": "icloud", "appName": "tg",
				"count": count, "updatedAt": "2026-09-10T14:51:26Z",
			})
		})
		stock, err := cli.GetEmailStock(context.Background(), "tg", EmailTypeICloud)
		if stock != nil || !errors.Is(err, ErrInvalidJSONResult) {
			t.Fatalf("accepted invalid count %v: %v", count, err)
		}
	}
}
