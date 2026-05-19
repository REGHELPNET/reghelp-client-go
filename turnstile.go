package reghelp

import "context"

// TurnstileRequest contains the parameters for [Client.GetTurnstileToken].
type TurnstileRequest struct {
	URL     string // page URL with the widget
	SiteKey string // Turnstile site key
	Action  string // optional expected action
	CData   string // optional custom data
	Proxy   string // optional, "scheme://host:port"
	Actor   string // optional
	Scope   string // optional
	Ref     string
	Webhook string
}

// GetTurnstileToken solves a Cloudflare Turnstile challenge.
func (c *Client) GetTurnstileToken(ctx context.Context, req TurnstileRequest) (*TokenResponse, error) {
	params := map[string]string{
		"url":     req.URL,
		"siteKey": req.SiteKey,
		"action":  req.Action,
		"cdata":   req.CData,
		"proxy":   req.Proxy,
		"actor":   req.Actor,
		"scope":   req.Scope,
		"ref":     req.Ref,
		"webHook": req.Webhook,
	}
	raw, err := c.do(ctx, "/turnstile/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTurnstileStatus polls a Turnstile task.
func (c *Client) GetTurnstileStatus(ctx context.Context, taskID string) (*TurnstileStatusResponse, error) {
	raw, err := c.do(ctx, "/turnstile/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &TurnstileStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
