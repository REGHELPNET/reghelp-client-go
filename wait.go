package reghelp

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// WaitForResult polls the matching getStatus endpoint until the task reaches
// a terminal state (done or error), the deadline elapses, or ctx is canceled.
//
// timeout 0 → 180s (matches Python SDK default).
// pollInterval 0 → 2s (matches Python SDK default).
//
// Returns the concrete *XxxStatusResponse (use AnyStatus for the interface
// boundary). On timeout returns an error matching ErrTimeout (use errors.Is);
// on ctx cancel returns ctx.Err().
//
// A temporary HTTP 503 from the status endpoint (errors.Is(err, ErrUnavailable))
// does not end the wait: polling continues until the deadline.
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
	// One poll may itself repeat a 503 and wait for Retry-After, so every poll
	// runs under the wait deadline, not only under the caller's ctx.
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	var lastUnavailable error
	timedOut := func() error {
		if lastUnavailable != nil {
			return fmt.Errorf("%w (last status read: %v)", ErrTimeout, lastUnavailable)
		}
		return ErrTimeout
	}

	for {
		pause := pollInterval
		st, err := c.pollStatus(pollCtx, service, taskID)
		switch {
		case err == nil:
			if st.getStatus().IsTerminal() {
				return st, nil
			}
			lastUnavailable = nil
		case ctx.Err() == nil && pollCtx.Err() != nil:
			// Our own deadline cut the poll short, not the caller's ctx.
			return nil, timedOut()
		case errors.Is(err, ErrUnavailable) && ctx.Err() == nil:
			// The status read failed, the task did not: keep polling until
			// the caller's own deadline instead of giving the task up.
			var apiErr *Error
			if errors.As(err, &apiErr) && apiErr.RetryAfter > pause {
				pause = apiErr.RetryAfter
			}
			lastUnavailable = err
		default:
			return nil, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, timedOut()
		}
		if pause > remaining {
			pause = remaining
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pause):
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
	case ServiceAttestation:
		return c.GetAttestationStatus(ctx, taskID)
	default:
		return nil, &Error{Code: "INVALID_PARAM", Message: "unknown service: " + string(service)}
	}
}
