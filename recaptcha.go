package reghelp

import "context"

// RecaptchaMobileRequest contains the parameters for
// [Client.GetRecaptchaMobileToken].
type RecaptchaMobileRequest struct {
	AppName   string    // e.g. "org.telegram.messenger"
	AppDevice AppDevice // iOS / Android
	AppKey    string    // reCAPTCHA site key
	AppAction string    // e.g. "login"
	Proxy     *ProxyConfig
	Ref       string
	Webhook   string
}

// GetRecaptchaMobileToken creates an asynchronous mobile reCAPTCHA task.
func (c *Client) GetRecaptchaMobileToken(ctx context.Context, req RecaptchaMobileRequest) (*TokenResponse, error) {
	params := map[string]string{
		"appName":   req.AppName,
		"appDevice": string(req.AppDevice),
		"appKey":    req.AppKey,
		"appAction": req.AppAction,
	}
	if req.Proxy != nil {
		if err := req.Proxy.Validate(); err != nil {
			return nil, &Error{Code: "INVALID_PARAM", Message: err.Error()}
		}
		req.Proxy.apply(params)
	}
	if req.Ref != "" {
		params["ref"] = req.Ref
	}
	if req.Webhook != "" {
		params["webHook"] = req.Webhook
	}

	raw, err := c.do(ctx, "/RecaptchaMobile/getToken", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &TokenResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRecaptchaMobileStatus polls a mobile reCAPTCHA task.
func (c *Client) GetRecaptchaMobileStatus(ctx context.Context, taskID string) (*RecaptchaMobileStatusResponse, error) {
	raw, err := c.do(ctx, "/RecaptchaMobile/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &RecaptchaMobileStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
