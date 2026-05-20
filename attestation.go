package reghelp

import (
	"context"
	"strconv"
)

// AttestationTokenRequest contains the parameters for
// [Client.GetAttestationToken].
//
// AuthKey is the only mandatory field — every other knob has a sensible
// default on the attestation-server side. The default flow targets
// WhatsApp Key Attestation: com.whatsapp packageName, embedded WhatsApp
// APK signature, current versionCode, zero verifiedBoot placeholder.
type AttestationTokenRequest struct {
	// AuthKey is the Google-issued challenge nonce.
	// Hex or base64 (std/url-safe), 4..512 characters.
	AuthKey string

	// Optional 32-byte hex overrides. Empty values fall back to the
	// server's zero-array placeholder (C3/C4 normalisation rule in
	// attestation-server).
	VerifiedBootKey  string
	VerifiedBootHash string

	// Optional embedded APK metadata overrides. Leave zero/empty to use
	// the attestation-server defaults (current WhatsApp build).
	APKVersionCode     int
	PackageName        string
	APKSignatureSha256 string

	// Optional base64 payload to ECDSA-sign with the leaf key. The
	// signature comes back as `Sign` in the status response.
	Enc string

	// Ref is an optional referral tag.
	Ref string
	// Webhook is an optional URL to receive completion notifications.
	Webhook string
}

// GetAttestationToken creates a WhatsApp Key Attestation task.
//
// The result is asynchronous: the returned TokenResponse carries the
// task ID; poll GetAttestationStatus (or WaitForResult with
// service="attestation") until Status == TaskStatusDone.
func (c *Client) GetAttestationToken(ctx context.Context, req AttestationTokenRequest) (*TokenResponse, error) {
	if l := len(req.AuthKey); l < 4 || l > 512 {
		return nil, &Error{Code: "INVALID_PARAM", Message: "AuthKey length must be 4..512"}
	}
	if req.APKVersionCode != 0 && (req.APKVersionCode < 1 || req.APKVersionCode > 2_147_483_647) {
		return nil, &Error{Code: "INVALID_PARAM", Message: "APKVersionCode must be in range 1..2_147_483_647"}
	}

	params := map[string]string{"authkey": req.AuthKey}
	if req.VerifiedBootKey != "" {
		params["verifiedBootKey"] = req.VerifiedBootKey
	}
	if req.VerifiedBootHash != "" {
		params["verifiedBootHash"] = req.VerifiedBootHash
	}
	if req.APKVersionCode != 0 {
		params["apkVersionCode"] = strconv.Itoa(req.APKVersionCode)
	}
	if req.PackageName != "" {
		params["packageName"] = req.PackageName
	}
	if req.APKSignatureSha256 != "" {
		params["apkSignatureSha256"] = req.APKSignatureSha256
	}
	if req.Enc != "" {
		params["enc"] = req.Enc
	}
	if req.Ref != "" {
		params["ref"] = req.Ref
	}
	if req.Webhook != "" {
		params["webHook"] = req.Webhook
	}

	raw, err := c.do(ctx, "/attestation/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAttestationStatus polls a WhatsApp Key Attestation task.
func (c *Client) GetAttestationStatus(ctx context.Context, taskID string) (*AttestationStatusResponse, error) {
	raw, err := c.do(ctx, "/attestation/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &AttestationStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

