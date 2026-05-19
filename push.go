package reghelp

import "context"

// PushTokenRequest contains the parameters for [Client.GetPushToken].
type PushTokenRequest struct {
	AppName    string    // e.g. "tgiOS", "tg", "tg_beta", "tg_x"
	AppDevice  AppDevice // AppDeviceIOS / AppDeviceAndroid
	AppVersion string    // optional
	AppBuild   string    // optional
	Ref        string    // optional referral tag
	Webhook    string    // optional notification URL
}

// GetPushToken creates an asynchronous task for an APNS / FCM push token.
func (c *Client) GetPushToken(ctx context.Context, req PushTokenRequest) (*TokenResponse, error) {
	params := map[string]string{
		"appName":    req.AppName,
		"appDevice":  string(req.AppDevice),
		"appVersion": req.AppVersion,
		"appBuild":   req.AppBuild,
		"ref":        req.Ref,
		"webHook":    req.Webhook,
	}
	raw, err := c.do(ctx, "/push/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPushStatus polls a push-token task.
func (c *Client) GetPushStatus(ctx context.Context, taskID string) (*PushStatusResponse, error) {
	raw, err := c.do(ctx, "/push/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &PushStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetPushStatusResult is the parsed envelope returned by SetPushStatus.
type SetPushStatusResult struct {
	Success bool
	// Balance is set when the server returned an updated balance on either
	// "success" or recoverable "error" envelopes (e.g. refund flow).
	Balance *float64
	// Raw is the original decoded JSON envelope, in case the caller needs more.
	Raw map[string]any
}

// SetPushStatus reports a failed push-token task to enable refund.
// Status is one of [PushStatusType] values (NOSMS, FLOOD, BANNED, 2FA).
// PhoneNumber must be in E.164 format.
func (c *Client) SetPushStatus(ctx context.Context, taskID, phoneNumber string, status PushStatusType) (*SetPushStatusResult, error) {
	params := map[string]string{
		"id":     taskID,
		"number": phoneNumber,
		"status": string(status),
	}
	raw, err := c.do(ctx, "/push/setStatus", params, taskID, true)
	if err != nil {
		return nil, err
	}
	res := &SetPushStatusResult{Raw: raw}
	if b, ok := raw["balance"].(float64); ok {
		bb := b
		res.Balance = &bb
	}
	switch asString(raw["status"]) {
	case "success":
		res.Success = true
	case "error":
		// Refund-with-balance is allowed by the server; surface as Success=true
		// when balance is present.
		if res.Balance != nil {
			res.Success = true
		}
	}
	return res, nil
}
