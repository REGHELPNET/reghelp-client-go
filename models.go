package reghelp

// BalanceResponse is the result of GetBalance.
type BalanceResponse struct {
	Status   string  `json:"status"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

// TokenResponse is the create-task response shared by Push / VoIP /
// Email / Integrity / Recaptcha / Turnstile getToken endpoints.
type TokenResponse struct {
	Status  string  `json:"status"`
	ID      string  `json:"id"`
	Service string  `json:"service"`
	Product string  `json:"product"`
	Price   float64 `json:"price"`
	Balance float64 `json:"balance"`
}

// EmailGetResponse is the result of GetEmail (the email is allocated immediately;
// the verification code is retrieved later via WaitForResult / GetEmailStatus).
type EmailGetResponse struct {
	Status  string  `json:"status"`
	ID      string  `json:"id"`
	Email   string  `json:"email"`
	Service string  `json:"service"`
	Product string  `json:"product"`
	Price   float64 `json:"price"`
	Balance float64 `json:"balance"`
}

// StatusResponse is the common shape of all *getStatus endpoints. Concrete
// token / code fields are populated only when Status == TaskStatusDone.
type StatusResponse struct {
	ID      string     `json:"id"`
	Status  TaskStatus `json:"status"`
	Message string     `json:"message,omitempty"`
}

// PushStatusResponse is the push.getStatus result.
type PushStatusResponse struct {
	StatusResponse
	Token string `json:"token,omitempty"`
}

// VoipStatusResponse is the pushVoip.getStatus result.
type VoipStatusResponse struct {
	StatusResponse
	Token string `json:"token,omitempty"`
}

// EmailStatusResponse is the email.getStatus result.
type EmailStatusResponse struct {
	StatusResponse
	Email string `json:"email,omitempty"`
	Code  string `json:"code,omitempty"`
}

// IntegrityStatusResponse is the integrity.getStatus result.
type IntegrityStatusResponse struct {
	StatusResponse
	Token string `json:"token,omitempty"`
}

// RecaptchaMobileStatusResponse is the RecaptchaMobile.getStatus result.
type RecaptchaMobileStatusResponse struct {
	StatusResponse
	Token string `json:"token,omitempty"`
}

// TurnstileStatusResponse is the turnstile.getStatus result.
type TurnstileStatusResponse struct {
	StatusResponse
	Token string `json:"token,omitempty"`
}

// AttestationStatusResponse is the attestation.getStatus result.
//
// On TaskStatusDone the full SignResponse payload is returned: an
// X.509 certificate chain (base64-encoded DER concatenation), the
// optional ECDSA signature over the request's enc field, the leaf
// private key (PKCS#8 / base64), and the keybox device id used for
// debugging which keybox served the request.
type AttestationStatusResponse struct {
	StatusResponse
	Authorization     string `json:"authorization,omitempty"`
	Sign              string `json:"sign,omitempty"`
	LeafPrivateKeyB64 string `json:"leafPrivateKeyB64,omitempty"`
	KeyboxDeviceID    string `json:"keyboxDeviceId,omitempty"`
}

// AnyStatus is the return type of WaitForResult — one of *PushStatusResponse,
// *VoipStatusResponse, *EmailStatusResponse, *IntegrityStatusResponse,
// *RecaptchaMobileStatusResponse, *TurnstileStatusResponse,
// *AttestationStatusResponse.
type AnyStatus interface {
	getStatus() TaskStatus
}

func (s *PushStatusResponse) getStatus() TaskStatus            { return s.Status }
func (s *VoipStatusResponse) getStatus() TaskStatus            { return s.Status }
func (s *EmailStatusResponse) getStatus() TaskStatus           { return s.Status }
func (s *IntegrityStatusResponse) getStatus() TaskStatus       { return s.Status }
func (s *RecaptchaMobileStatusResponse) getStatus() TaskStatus { return s.Status }
func (s *TurnstileStatusResponse) getStatus() TaskStatus       { return s.Status }
func (s *AttestationStatusResponse) getStatus() TaskStatus     { return s.Status }
