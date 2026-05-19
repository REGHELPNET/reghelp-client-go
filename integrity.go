package reghelp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// IntegrityTokenRequest contains the parameters for [Client.GetIntegrityToken].
//
// AppVersionCode is mandatory since Key API v2026-05 — it must match the APK
// versionCode of the target app at signing time. Valid range: 1..2_147_483_647.
//
// TokenType selects the Play Integrity flow:
//   - "" or [IntegrityTokenTypeClassic] — Classic (MEETS_STRONG_INTEGRITY, ~1-3s);
//     the `type` parameter is omitted on the wire.
//   - [IntegrityTokenTypeStd] — Standard/Express (MEETS_DEVICE_INTEGRITY,
//     ~200-600ms); sent as `type=std`.
type IntegrityTokenRequest struct {
	AppName        string
	AppDevice      AppDevice
	Nonce          string // URL-safe Base64, 16..500 chars
	AppVersionCode int    // 1..2_147_483_647 (mandatory)
	TokenType      IntegrityTokenType
	Ref            string
	Webhook        string
}

// GetIntegrityToken creates a Play Integrity attestation task.
func (c *Client) GetIntegrityToken(ctx context.Context, req IntegrityTokenRequest) (*TokenResponse, error) {
	if req.AppVersionCode < 1 || req.AppVersionCode > 2_147_483_647 {
		return nil, &Error{Code: "INVALID_PARAM", Message: "AppVersionCode must be in range 1..2_147_483_647"}
	}
	if l := len(req.Nonce); l < 16 || l > 500 {
		return nil, &Error{Code: "INVALID_PARAM", Message: fmt.Sprintf("Nonce length must be 16..500, got %d", l)}
	}

	params := map[string]string{
		"appName":        req.AppName,
		"appDevice":      string(req.AppDevice),
		"nonce":          req.Nonce,
		"appVersionCode": strconv.Itoa(req.AppVersionCode),
	}
	// Server default is Classic; only forward `type` for the STD/express flow.
	switch strings.ToLower(string(req.TokenType)) {
	case "std", "standard", "express":
		params["type"] = "std"
	case "", "classic", "default":
		// omit
	}
	if req.Ref != "" {
		params["ref"] = req.Ref
	}
	if req.Webhook != "" {
		params["webHook"] = req.Webhook
	}

	raw, err := c.do(ctx, "/integrity/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetIntegrityStatus polls a Play Integrity task.
func (c *Client) GetIntegrityStatus(ctx context.Context, taskID string) (*IntegrityStatusResponse, error) {
	raw, err := c.do(ctx, "/integrity/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &IntegrityStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
