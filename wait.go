package reghelp

import (
	"context"
	"time"
)

// WaitForResult polls the matching getStatus endpoint until the task reaches
// a terminal state (done or error), the deadline elapses, or ctx is canceled.
//
// timeout 0 → 180s (matches Python SDK default).
// pollInterval 0 → 2s (matches Python SDK default).
//
// Returns the concrete *XxxStatusResponse (use AnyStatus for the interface
// boundary). On timeout returns ErrTimeout; on ctx cancel returns ctx.Err().
func (c *Client) WaitForResult(
	ctx context.Context,
	taskID string,
	service Service,
	timeout time.Duration,
	pollInterval time.Duration,
) (AnyStatus, error) {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)

	for {
		st, err := c.pollStatus(ctx, service, taskID)
		if err != nil {
			return nil, err
		}
		if st.getStatus().IsTerminal() {
			return st, nil
		}
		if time.Until(deadline) <= 0 {
			return nil, ErrTimeout
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func (c *Client) pollStatus(ctx context.Context, service Service, taskID string) (AnyStatus, error) {
	switch service {
	case ServicePush:
		return c.GetPushStatus(ctx, taskID)
	case ServiceEmail:
		return c.GetEmailStatus(ctx, taskID)
	case ServiceIntegrity:
		return c.GetIntegrityStatus(ctx, taskID)
	case ServiceRecaptcha:
		return c.GetRecaptchaMobileStatus(ctx, taskID)
	case ServiceTurnstile:
		return c.GetTurnstileStatus(ctx, taskID)
	case ServiceVoIP:
		return c.GetVoIPStatus(ctx, taskID)
	default:
		return nil, &Error{Code: "INVALID_PARAM", Message: "unknown service: " + string(service)}
	}
}
