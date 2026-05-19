package reghelp

// TaskStatus is the lifecycle state of an asynchronous REGHelp task.
type TaskStatus string

const (
	TaskStatusWait       TaskStatus = "wait"
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusSubmitted  TaskStatus = "submitted"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusError      TaskStatus = "error"
)

// IsTerminal reports whether the task has reached a final state.
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusDone || s == TaskStatusError
}

// ProxyType identifies a proxy scheme accepted by ProxyConfig.
type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSOCKS4 ProxyType = "socks4"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

// AppDevice identifies the target mobile platform of a request.
type AppDevice string

const (
	AppDeviceIOS     AppDevice = "iOS"
	AppDeviceAndroid AppDevice = "Android"
)

// EmailType selects the temporary email provider.
type EmailType string

const (
	EmailTypeICloud EmailType = "icloud"
	EmailTypeGmail  EmailType = "gmail"
)

// IntegrityTokenType selects the Play Integrity flow.
//
// CLASSIC is the default and is sent on the wire by omitting the `type`
// parameter (returns a long-lived strong-integrity token, ~1-3s).
// STD opts into the Standard/Express flow with `type=std`
// (returns a device-integrity token, ~200-600ms).
type IntegrityTokenType string

const (
	IntegrityTokenTypeClassic IntegrityTokenType = "classic"
	IntegrityTokenTypeStd     IntegrityTokenType = "std"
)

// PushStatusType is the failure reason for SetPushStatus refund flow.
type PushStatusType string

const (
	PushStatusTypeNoSMS  PushStatusType = "NOSMS"
	PushStatusTypeFlood  PushStatusType = "FLOOD"
	PushStatusTypeBanned PushStatusType = "BANNED"
	PushStatusType2FA    PushStatusType = "2FA"
)

// Service identifies a polled service for WaitForResult.
type Service string

const (
	ServicePush      Service = "push"
	ServiceEmail     Service = "email"
	ServiceIntegrity Service = "integrity"
	ServiceRecaptcha Service = "recaptcha"
	ServiceTurnstile Service = "turnstile"
	ServiceVoIP      Service = "voip"
)
