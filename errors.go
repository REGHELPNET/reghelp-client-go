package reghelp

import (
	"errors"
	"fmt"
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
	case ErrExternalService:
		return e.Status == 502 || e.Code == "EXTERNAL_ERROR"
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
		case "EXTERNAL_ERROR":
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
