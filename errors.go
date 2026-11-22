package reghelp

import (
	"errors"
	"fmt"
	"time"
)

// Error is the base error type returned by the SDK.
//
// Status is the HTTP status code (or 0 for transport errors).
// Code mirrors the `id` field from the REGHelp error envelope when present
// (RATE_LIMIT, SERVICE_DISABLED, MAINTENANCE_MODE, TASK_NOT_FOUND,
// INVALID_PARAM, EXTERNAL_ERROR …).
// Raw holds the decoded error envelope for callers that need the original
// fields (e.g. balance on push.setStatus error).
type Error struct {
	Status  int
	Code    string
	Message string
	Raw     map[string]any
	// Unwrapped underlying error, if any (e.g. network/timeout).
	Cause error
	// RetryAfter is the server's Retry-After hint on a temporary HTTP 503
	// (zero when the header was absent or not given in seconds).
	RetryAfter time.Duration
	// repeatable is set only for a temporary 503 on a read-only status poll.
	repeatable bool
}

func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("[%d %s] %s", e.Status, e.Code, e.Message)
	}
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

// Sentinel error codes. Compare with errors.Is(err, reghelp.ErrUnauthorized).
var (
	ErrUnauthorized      = errors.New("reghelp: invalid api key")
	ErrRateLimit         = errors.New("reghelp: rate limit exceeded")
	ErrServiceDisabled   = errors.New("reghelp: service temporarily disabled")
	ErrMaintenance       = errors.New("reghelp: api in maintenance mode")
	ErrTaskNotFound      = errors.New("reghelp: task not found")
	ErrInvalidParameter  = errors.New("reghelp: invalid parameter")
	ErrExternalService   = errors.New("reghelp: external service error")
	ErrUnknown           = errors.New("reghelp: unknown error")
	ErrNetwork           = errors.New("reghelp: network error")
	ErrTimeout           = errors.New("reghelp: timeout waiting for task result")
	ErrInvalidJSONResult = errors.New("reghelp: invalid JSON in response")
)

// ErrUnavailable marks a temporary HTTP 503 on a read-only status poll
// (Get*Status / WaitForResult): the status could not be read and the same
// request can be repeated. A disabled service, maintenance mode, and a 503 on
// any other request (where a repeat is not known to be safe) are not
// ErrUnavailable.
var ErrUnavailable = errors.New("reghelp: service temporarily unavailable")

// Is reports whether err equals one of the sentinel errors above.
// Implements errors.Is for chained Error values.
func (e *Error) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == 401 || e.Code == "UNAUTHORIZED"
	case ErrRateLimit:
		return e.Status == 429 || e.Code == "RATE_LIMIT"
	case ErrServiceDisabled:
		return e.Code == "SERVICE_DISABLED"
	case ErrMaintenance:
		return e.Code == "MAINTENANCE_MODE"
	case ErrTaskNotFound:
		return e.Status == 404 || e.Code == "TASK_NOT_FOUND"
	case ErrInvalidParameter:
		return e.Status == 400 || e.Code == "INVALID_PARAM"
	case ErrUnavailable:
		return e.Status == 503 && e.repeatable
	case ErrExternalService:
		return e.Status == 502 || e.Code == "EXTERNAL_ERROR" || e.Code == "UPSTREAM_ERROR"
	}
	return false
}

// mapErrorCode converts the server `id`/HTTP-status pair to a typed Error.
// task is included in the message when relevant (TASK_NOT_FOUND).
func mapErrorCode(status int, code, message, task string, raw map[string]any) *Error {
	e := &Error{Status: status, Code: code, Message: message, Raw: raw}
	if e.Message == "" {
		switch code {
		case "RATE_LIMIT":
			e.Message = "rate limit exceeded"
		case "SERVICE_DISABLED":
			e.Message = "service temporarily disabled"
		case "MAINTENANCE_MODE":
			e.Message = "api in maintenance mode"
		case "TASK_NOT_FOUND":
			if task != "" {
				e.Message = "task " + task + " not found"
			} else {
				e.Message = "task not found"
			}
		case "INVALID_PARAM":
			e.Message = "invalid parameter"
		case "EXTERNAL_ERROR", "UPSTREAM_ERROR":
			e.Message = "external service error"
		case "UNAUTHORIZED":
			e.Message = "invalid api key"
		default:
			if code != "" {
				e.Message = "unknown error: " + code
			} else if status != 0 {
				e.Message = fmt.Sprintf("http %d", status)
			} else {
				e.Message = "unknown error"
			}
		}
	}
	return e
}
