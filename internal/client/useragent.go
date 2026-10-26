package client

import (
	"fmt"
	"net/http"
	"strings"
)

// userAgentTransport is an http.RoundTripper that stamps a User-Agent header on
// every outbound request. It wraps an existing RoundTripper so the underlying
// transport's behaviour (timeouts, proxies, TLS) is preserved.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The RoundTripper contract forbids mutating the caller's request, so clone
	// it before setting the header.
	r := req.Clone(req.Context())
	r.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(r)
}

// withUserAgent installs a userAgentTransport on the given client so all of its
// requests carry the supplied User-Agent. It is a no-op when ua is empty, which
// keeps unit tests that construct clients without a configured UA unaffected.
func withUserAgent(hc *http.Client, ua string) {
	if hc == nil || strings.TrimSpace(ua) == "" {
		return
	}
	base := hc.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	hc.Transport = &userAgentTransport{userAgent: ua, base: base}
}

// UserAgent builds the User-Agent string sent on every Anypoint API request so
// traffic can be attributed to this provider (and its version) plus the
// Terraform core version driving it, mirroring how the AWS/GCP providers
// identify themselves. terraformVersion may be empty (e.g. in unit tests or
// older Terraform cores that do not report it).
func UserAgent(providerVersion, terraformVersion string) string {
	if providerVersion == "" {
		providerVersion = "dev"
	}
	ua := fmt.Sprintf("terraform-provider-anypoint/%s (+https://registry.terraform.io/providers/mulesoft/anypoint)", providerVersion)
	if terraformVersion != "" {
		ua += " Terraform/" + terraformVersion
	}
	return ua
}
