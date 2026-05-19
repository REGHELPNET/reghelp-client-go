package reghelp

import "context"

// VoIPTokenRequest contains the parameters for [Client.GetVoIPToken] (iOS).
type VoIPTokenRequest struct {
	AppName string
	Ref     string // optional
	Webhook string // optional
}

// GetVoIPToken creates an asynchronous task for an iOS VoIP push token.
// On iOS both push (APNS) and VoIP tokens are required for registerDevice.
func (c *Client) GetVoIPToken(ctx context.Context, req VoIPTokenRequest) (*TokenResponse, error) {
	params := map[string]string{
		"appName": req.AppName,
		"ref":     req.Ref,
		"webHook": req.Webhook,
	}
	raw, err := c.do(ctx, "/pushVoip/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetVoIPStatus polls a VoIP token task.
func (c *Client) GetVoIPStatus(ctx context.Context, taskID string) (*VoipStatusResponse, error) {
	raw, err := c.do(ctx, "/pushVoip/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &VoipStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
