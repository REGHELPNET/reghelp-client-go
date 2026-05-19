package reghelp

import "context"

// EmailRequest contains the parameters for [Client.GetEmail].
type EmailRequest struct {
	AppName   string
	AppDevice AppDevice
	Phone     string // E.164
	Type      EmailType
	Ref       string // optional
	Webhook   string // optional
}

// GetEmail allocates a temporary email address (iCloud HME or Gmail OAuth) for
// the given phone. The verification code is delivered asynchronously — poll
// via [Client.WaitForResult] / [Client.GetEmailStatus].
func (c *Client) GetEmail(ctx context.Context, req EmailRequest) (*EmailGetResponse, error) {
	params := map[string]string{
		"appName":   req.AppName,
		"appDevice": string(req.AppDevice),
		"phone":     req.Phone,
		"type":      string(req.Type),
		"ref":       req.Ref,
		"webHook":   req.Webhook,
	}
	raw, err := c.do(ctx, "/email/getEmail", params, "", false)
	if err != nil {
		return nil, err
	}
	out := &EmailGetResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEmailStatus polls an email-task for the verification code.
func (c *Client) GetEmailStatus(ctx context.Context, taskID string) (*EmailStatusResponse, error) {
	raw, err := c.do(ctx, "/email/getStatus", map[string]string{"id": taskID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &EmailStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
