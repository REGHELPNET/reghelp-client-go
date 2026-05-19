package reghelp

import "fmt"

// ProxyConfig describes an upstream proxy for services that support it
// (currently only RecaptchaMobile via proxyType/proxyAddress/...).
//
// Address up to 255 chars, Login up to 128, Password up to 256 (limits
// enforced by the REGHelp API since v1.2.1).
type ProxyConfig struct {
	Type     ProxyType
	Address  string
	Port     int
	Login    string // optional
	Password string // optional
}

// Validate reports whether p is structurally valid for an outgoing request.
func (p ProxyConfig) Validate() error {
	if p.Type == "" {
		return fmt.Errorf("reghelp: proxy.Type required")
	}
	if p.Address == "" {
		return fmt.Errorf("reghelp: proxy.Address required")
	}
	if len(p.Address) > 255 {
		return fmt.Errorf("reghelp: proxy.Address exceeds 255 chars")
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("reghelp: proxy.Port out of range (1..65535)")
	}
	if len(p.Login) > 128 {
		return fmt.Errorf("reghelp: proxy.Login exceeds 128 chars")
	}
	if len(p.Password) > 256 {
		return fmt.Errorf("reghelp: proxy.Password exceeds 256 chars")
	}
	return nil
}

// apply spreads the proxy fields into a query-param map using the
// REGHelp-style camelCase keys (proxyType / proxyAddress / proxyPort
// / proxyLogin / proxyPassword).
func (p ProxyConfig) apply(params map[string]string) {
	params["proxyType"] = string(p.Type)
	params["proxyAddress"] = p.Address
	params["proxyPort"] = fmt.Sprintf("%d", p.Port)
	if p.Login != "" {
		params["proxyLogin"] = p.Login
	}
	if p.Password != "" {
		params["proxyPassword"] = p.Password
	}
}
