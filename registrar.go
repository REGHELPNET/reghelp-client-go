package reghelp

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// Provider registrar flow (WhatsApp, Snapchat, …).
//
// These endpoints bind a server-owned Android profile to a client-owned
// registrar session, then create Play Integrity / Key Attestation tasks on
// that pinned profile so every artifact for one registration attempt comes
// from the SAME device instance. Mirrors the Python SDK's provider-generic
// registrar methods. Provider is the namespace prefix on the wire, e.g.
// "whatsapp" -> /whatsapp/registrar/start.

var providerRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// providerEndpoint builds a provider-scoped path (/{provider}/{endpoint})
// without permitting path injection.
func providerEndpoint(provider, endpoint string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if !providerRe.MatchString(p) {
		return "", &Error{Code: "INVALID_PARAM", Message: "provider must match ^[a-z][a-z0-9_-]{0,63}$"}
	}
	return "/" + p + "/" + trimLeftSlash(endpoint), nil
}

// idempotencyHeaders builds the Idempotency-Key header for paid registrar task
// creation. Returns nil when requestID is empty — the Key API treats a missing
// header as "no idempotency". Never auto-generate a key: doing so makes the
// backend dedupe the create and the task is silently not created.
func idempotencyHeaders(requestID string) map[string]string {
	id := strings.TrimSpace(requestID)
	if id == "" {
		return nil
	}
	return map[string]string{"Idempotency-Key": id}
}

func requireText(v, name string) error {
	if strings.TrimSpace(v) == "" {
		return &Error{Code: "INVALID_PARAM", Message: name + " must not be empty"}
	}
	return nil
}

func validVersionCode(v int) error {
	if v < 1 || v > 2_147_483_647 {
		return &Error{Code: "INVALID_PARAM", Message: "appVersionCode must be in range 1..2_147_483_647"}
	}
	return nil
}

// ── Response models ─────────────────────────────────────────────────────────

// AppParamsResponse holds server-maintained opaque application metadata for a
// registrar provider flavor.
type AppParamsResponse struct {
	Status                  string         `json:"status,omitempty"`
	App                     string         `json:"app,omitempty"`
	Params                  map[string]any `json:"params,omitempty"`
	NativeFieldsStale       *bool          `json:"nativeFieldsStale,omitempty"`
	NativeFieldsVersionCode *int           `json:"nativeFieldsVersionCode,omitempty"`
	VersionCode             *int           `json:"versionCode,omitempty"`
	VersionName             string         `json:"versionName,omitempty"`
}

// RegistrarBindingResponse describes the Android profile bound to one
// client-owned registrar session. An unknown session is a normal, readable
// outcome: Status == "not_found" with HTTP 200 (branch on Status, don't treat
// it as an error).
type RegistrarBindingResponse struct {
	Status             string         `json:"status"`
	Provider           string         `json:"provider,omitempty"`
	AppName            string         `json:"appName,omitempty"`
	RegistrarSessionID string         `json:"registrarSessionId,omitempty"`
	ProfileID          string         `json:"profile_id,omitempty"`
	DeviceProfile      map[string]any `json:"device_profile,omitempty"`
	Message            string         `json:"message,omitempty"`
	Detail             string         `json:"detail,omitempty"`
}

// BoundArtifactTaskResponse is the create-task response for a task accepted
// through a provider registrar binding (integrity or attestation).
type BoundArtifactTaskResponse struct {
	ID      string   `json:"id"`
	Status  string   `json:"status,omitempty"`
	Service string   `json:"service,omitempty"`
	Product string   `json:"product,omitempty"`
	Price   *float64 `json:"price,omitempty"`
	Balance *float64 `json:"balance,omitempty"`
}

// BoundIntegrityStatusResponse is a Play Integrity result tied to a binding.
type BoundIntegrityStatusResponse struct {
	ID                 string         `json:"id,omitempty"`
	Status             TaskStatus     `json:"status"`
	Token              string         `json:"token,omitempty"`
	RegistrarSessionID string         `json:"registrarSessionId,omitempty"`
	ProfileID          string         `json:"profile_id,omitempty"`
	DeviceProfile      map[string]any `json:"device_profile,omitempty"`
	Message            string         `json:"message,omitempty"`
	Detail             string         `json:"detail,omitempty"`
	Error              string         `json:"error,omitempty"`
	Reason             string         `json:"reason,omitempty"`
}

// BoundAttestationStatusResponse is an Android Key Attestation result tied to a
// binding. On TaskStatusDone the SignResponse payload is populated
// (Authorization = base64 chain DER, optional Sign, leaf private key, keybox id).
type BoundAttestationStatusResponse struct {
	ID                 string         `json:"id,omitempty"`
	Status             TaskStatus     `json:"status"`
	Authorization      string         `json:"authorization,omitempty"`
	Sign               string         `json:"sign,omitempty"`
	LeafPrivateKeyB64  string         `json:"leafPrivateKeyB64,omitempty"`
	KeyboxDeviceID     string         `json:"keyboxDeviceId,omitempty"`
	RegistrarSessionID string         `json:"registrarSessionId,omitempty"`
	ProfileID          string         `json:"profile_id,omitempty"`
	DeviceProfile      map[string]any `json:"device_profile,omitempty"`
	Message            string         `json:"message,omitempty"`
	Detail             string         `json:"detail,omitempty"`
	Error              string         `json:"error,omitempty"`
	Reason             string         `json:"reason,omitempty"`
}

// ── Methods ─────────────────────────────────────────────────────────────────

// GetProviderAppParams fetches server-maintained opaque app parameters for a
// provider flavor (e.g. provider="whatsapp", appName="wa").
func (c *Client) GetProviderAppParams(ctx context.Context, provider, appName string) (*AppParamsResponse, error) {
	ep, err := providerEndpoint(provider, "/appParams")
	if err != nil {
		return nil, err
	}
	if err := requireText(appName, "appName"); err != nil {
		return nil, err
	}
	raw, err := c.do(ctx, ep, map[string]string{"app": appName}, "", false)
	if err != nil {
		return nil, err
	}
	out := &AppParamsResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// StartRegistrarRequest are the parameters for [Client.StartRegistrar].
type StartRegistrarRequest struct {
	Provider           string
	RegistrarSessionID string
	AppName            string
	AppVersionCode     int
	// RequestID, when set, is sent as Idempotency-Key so replaying the same
	// call returns the original binding instead of creating a second one.
	RequestID string
}

// StartRegistrar idempotently binds an Android profile to a client-owned
// registrar session.
func (c *Client) StartRegistrar(ctx context.Context, req StartRegistrarRequest) (*RegistrarBindingResponse, error) {
	ep, err := providerEndpoint(req.Provider, "/registrar/start")
	if err != nil {
		return nil, err
	}
	if err := requireText(req.RegistrarSessionID, "RegistrarSessionID"); err != nil {
		return nil, err
	}
	if err := requireText(req.AppName, "AppName"); err != nil {
		return nil, err
	}
	if err := validVersionCode(req.AppVersionCode); err != nil {
		return nil, err
	}
	params := map[string]string{
		"registrarSessionId": req.RegistrarSessionID,
		"appName":            req.AppName,
		"appVersionCode":     strconv.Itoa(req.AppVersionCode),
	}
	raw, err := c.doH(ctx, ep, params, idempotencyHeaders(req.RequestID), "", false)
	if err != nil {
		return nil, err
	}
	out := &RegistrarBindingResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRegistrarBinding reads the profile binding for a client-owned registrar
// session. A missing session returns Status == "not_found" (HTTP 200), not an
// error — branch on the Status field.
func (c *Client) GetRegistrarBinding(ctx context.Context, provider, registrarSessionID string) (*RegistrarBindingResponse, error) {
	ep, err := providerEndpoint(provider, "/registrar/getBinding")
	if err != nil {
		return nil, err
	}
	if err := requireText(registrarSessionID, "registrarSessionID"); err != nil {
		return nil, err
	}
	raw, err := c.do(ctx, ep, map[string]string{"registrarSessionId": registrarSessionID}, "", true)
	if err != nil {
		return nil, err
	}
	out := &RegistrarBindingResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BoundIntegrityTokenRequest are the parameters for
// [Client.GetBoundIntegrityToken].
type BoundIntegrityTokenRequest struct {
	Provider           string
	RegistrarSessionID string
	AppName            string
	Nonce              string // URL-safe Base64, 16..500 chars
	AppVersionCode     int
	// TokenType: "" omits the type (server default), [IntegrityTokenTypeStd]
	// sends type=std, [IntegrityTokenTypeClassic] sends type=classic.
	TokenType IntegrityTokenType
	RequestID string
}

// GetBoundIntegrityToken creates a Play Integrity task on the registrar
// session's bound profile.
func (c *Client) GetBoundIntegrityToken(ctx context.Context, req BoundIntegrityTokenRequest) (*BoundArtifactTaskResponse, error) {
	ep, err := providerEndpoint(req.Provider, "/integrity/getToken")
	if err != nil {
		return nil, err
	}
	if err := requireText(req.RegistrarSessionID, "RegistrarSessionID"); err != nil {
		return nil, err
	}
	if err := requireText(req.AppName, "AppName"); err != nil {
		return nil, err
	}
	if l := len(req.Nonce); l < 16 || l > 500 {
		return nil, &Error{Code: "INVALID_PARAM", Message: "Nonce length must be 16..500"}
	}
	if err := validVersionCode(req.AppVersionCode); err != nil {
		return nil, err
	}
	params := map[string]string{
		"registrarSessionId": req.RegistrarSessionID,
		"appName":            req.AppName,
		"nonce":              req.Nonce,
		"appVersionCode":     strconv.Itoa(req.AppVersionCode),
	}
	switch strings.ToLower(string(req.TokenType)) {
	case "std", "standard", "express":
		params["type"] = "std"
	case "classic", "default":
		params["type"] = "classic"
	case "":
		// omit — server default
	}
	raw, err := c.doH(ctx, ep, params, idempotencyHeaders(req.RequestID), "", false)
	if err != nil {
		return nil, err
	}
	out := &BoundArtifactTaskResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBoundIntegrityStatus polls a bound Play Integrity task.
func (c *Client) GetBoundIntegrityStatus(ctx context.Context, provider, taskID, registrarSessionID string) (*BoundIntegrityStatusResponse, error) {
	ep, err := providerEndpoint(provider, "/integrity/getStatus")
	if err != nil {
		return nil, err
	}
	if err := requireText(taskID, "taskID"); err != nil {
		return nil, err
	}
	if err := requireText(registrarSessionID, "registrarSessionID"); err != nil {
		return nil, err
	}
	raw, err := c.do(ctx, ep, map[string]string{"id": taskID, "registrarSessionId": registrarSessionID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &BoundIntegrityStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// BoundAttestationTokenRequest are the parameters for
// [Client.GetBoundAttestationToken].
type BoundAttestationTokenRequest struct {
	Provider           string
	RegistrarSessionID string
	AuthKey            string
	AppName            string
	AppVersionCode     int
	APKSignatureSha256 string // 64-hex; required by the registrar attestation flow
	Enc                string // optional base64 payload to ECDSA-sign with the leaf key
	RequestID          string
}

// GetBoundAttestationToken creates a Key Attestation task on the registrar
// session's bound profile.
func (c *Client) GetBoundAttestationToken(ctx context.Context, req BoundAttestationTokenRequest) (*BoundArtifactTaskResponse, error) {
	ep, err := providerEndpoint(req.Provider, "/attestation/getToken")
	if err != nil {
		return nil, err
	}
	if err := requireText(req.RegistrarSessionID, "RegistrarSessionID"); err != nil {
		return nil, err
	}
	if err := requireText(req.AuthKey, "AuthKey"); err != nil {
		return nil, err
	}
	if err := requireText(req.AppName, "AppName"); err != nil {
		return nil, err
	}
	if err := requireText(req.APKSignatureSha256, "APKSignatureSha256"); err != nil {
		return nil, err
	}
	if err := validVersionCode(req.AppVersionCode); err != nil {
		return nil, err
	}
	params := map[string]string{
		"registrarSessionId": req.RegistrarSessionID,
		"authkey":            req.AuthKey,
		"appName":            req.AppName,
		"appVersionCode":     strconv.Itoa(req.AppVersionCode),
		"apkSignatureSha256": req.APKSignatureSha256,
	}
	if req.Enc != "" {
		params["enc"] = req.Enc
	}
	raw, err := c.doH(ctx, ep, params, idempotencyHeaders(req.RequestID), "", false)
	if err != nil {
		return nil, err
	}
	out := &BoundArtifactTaskResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBoundAttestationStatus polls a bound Key Attestation task.
func (c *Client) GetBoundAttestationStatus(ctx context.Context, provider, taskID, registrarSessionID string) (*BoundAttestationStatusResponse, error) {
	ep, err := providerEndpoint(provider, "/attestation/getStatus")
	if err != nil {
		return nil, err
	}
	if err := requireText(taskID, "taskID"); err != nil {
		return nil, err
	}
	if err := requireText(registrarSessionID, "registrarSessionID"); err != nil {
		return nil, err
	}
	raw, err := c.do(ctx, ep, map[string]string{"id": taskID, "registrarSessionId": registrarSessionID}, taskID, true)
	if err != nil {
		return nil, err
	}
	out := &BoundAttestationStatusResponse{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
