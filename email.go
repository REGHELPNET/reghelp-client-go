package reghelp

import (
	"context"
	"fmt"
)

// GetEmailStock reads available address count for an application code (tg, ig,
// wa) or supported alias. It does not allocate a task, reserve an address or
// charge balance. Disabled services return ErrServiceDisabled; unavailable or
// stale stock returns ErrExternalService. A successful count of zero is valid.
func (c *Client) GetEmailStock(ctx context.Context, appName string, emailType EmailType) (*EmailStockResponse, error) {
	raw, err := c.do(ctx, "/email/getStock", map[string]string{
		"appName": appName, "type": string(emailType),
	}, "", false)
	if err != nil {
		return nil, err
	}
	out := &EmailStockResponse{}
	if err := decode(raw, out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSONResult, err)
	}
	if raw["count"] == nil || out.Count < 0 || out.UpdatedAt.IsZero() || out.Status != "success" || out.AppName == "" || (out.Service != EmailTypeICloud && out.Service != EmailTypeGmail) {
		return nil, fmt.Errorf("%w: incomplete or invalid email stock", ErrInvalidJSONResult)
	}
	return out, nil
}

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
