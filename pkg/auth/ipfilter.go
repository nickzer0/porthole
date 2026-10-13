package auth

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// OriginSecretHeader carries the shared secret CloudFront injects as a custom
// origin header. Because CloudFront overwrites any same-named header sent by a
// viewer, its presence proves the request arrived through the distribution.
const OriginSecretHeader = "X-Porthole-Origin"

// IPFilter restricts access by client IP.
//
// The client IP is only as trustworthy as the hop that reported it. Requests
// carrying a valid origin secret came through CloudFront, so CloudFront's view
// of the viewer is authoritative. Everything else arrived at the ngrok URL
// directly, where any header can be forged — which is why requireCloudFront
// rejects those outright once a distribution is in play.
type IPFilter struct {
	allowed           []*net.IPNet
	enabled           bool
	requireCloudFront bool
	originSecret      string
}

// NewIPFilter builds a filter from a list of IPs and CIDRs. Bare addresses are
// treated as single hosts. An empty list disables IP restriction entirely.
// Malformed entries are an error rather than a silent skip: a typo in the
// allowlist must not quietly fail open.
func NewIPFilter(entries []string, originSecret string) (*IPFilter, error) {
	f := &IPFilter{originSecret: originSecret}

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q in allowlist: %w", entry, err)
			}
			f.allowed = append(f.allowed, network)
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address %q in allowlist", entry)
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		f.allowed = append(f.allowed, &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(bits, bits),
		})
	}

	f.enabled = len(f.allowed) > 0
	return f, nil
}

// Enabled reports whether any IP restriction is in force.
func (f *IPFilter) Enabled() bool {
	return f.enabled
}

// RequireCloudFront makes the filter reject any non-loopback request that
// doesn't carry the origin secret, which closes the CDN bypass: the ngrok URL
// stops serving anything. Only turn this on once the secret is actually live
// on the distribution.
func (f *IPFilter) RequireCloudFront() {
	f.requireCloudFront = true
}

// viaCloudFront reports whether the request carries the origin secret.
func (f *IPFilter) viaCloudFront(r *http.Request) bool {
	if f.originSecret == "" {
		return false
	}
	got := r.Header.Get(OriginSecretHeader)
	return subtle.ConstantTimeCompare([]byte(got), []byte(f.originSecret)) == 1
}

// forwardedFor returns the X-Forwarded-For entries, left to right.
func forwardedFor(r *http.Request) []string {
	raw := r.Header.Get("X-Forwarded-For")
	if raw == "" {
		return nil
	}

	var entries []string
	for _, entry := range strings.Split(raw, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

// clientIP resolves the client address for the request.
//
// X-Forwarded-For is always read from the right. Each hop appends the peer it
// saw, so anything a client forges can only ever sit to the left of the
// entries added by infrastructure we control.
func (f *IPFilter) clientIP(r *http.Request) net.IP {
	xff := forwardedFor(r)

	if f.viaCloudFront(r) {
		// CloudFront reports the viewer directly when the origin request
		// policy forwards this header. It is IP:port.
		if addr := r.Header.Get("CloudFront-Viewer-Address"); addr != "" {
			if host, _, err := net.SplitHostPort(addr); err == nil {
				if ip := net.ParseIP(host); ip != nil {
					return ip
				}
			}
		}

		// Otherwise: CloudFront appends the viewer IP, then ngrok appends the
		// CloudFront edge IP it saw, giving "..., <viewer>, <cf-edge>".
		if len(xff) >= 2 {
			if ip := net.ParseIP(xff[len(xff)-2]); ip != nil {
				return ip
			}
		}
	}

	if len(xff) > 0 {
		if ip := net.ParseIP(xff[len(xff)-1]); ip != nil {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

// isLocalRequest reports whether the request came from a browser on this
// machine, rather than from the internet. Local access is never blocked, so a
// bad allowlist can't lock you out of your own box.
//
// A loopback RemoteAddr is NOT sufficient on its own: the ngrok agent runs on
// this machine and forwards to localhost, so every tunnelled request also
// arrives from 127.0.0.1. Tunnelled requests are distinguished by the proxy
// headers the hops in front of us add — a genuine local request has neither.
// Clients can only add such headers, never suppress the ones ngrok appends.
func isLocalRequest(r *http.Request) bool {
	if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get(OriginSecretHeader) != "" {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// allow decides whether the request may proceed, returning the resolved client
// IP and a reason when it may not.
func (f *IPFilter) allow(r *http.Request, checkAllowlist bool) (net.IP, string, bool) {
	if isLocalRequest(r) {
		return nil, "", true
	}

	if f.requireCloudFront && !f.viaCloudFront(r) {
		return nil, "request did not arrive via CloudFront", false
	}

	if !checkAllowlist || !f.enabled {
		return nil, "", true
	}

	ip := f.clientIP(r)
	if ip == nil {
		return nil, "client IP could not be determined", false
	}

	for _, network := range f.allowed {
		if network.Contains(ip) {
			return ip, "", true
		}
	}

	return ip, "IP not in allowlist", false
}

func (f *IPFilter) deny(w http.ResponseWriter, r *http.Request, ip net.IP, reason string) {
	from := "unknown"
	if ip != nil {
		from = ip.String()
	}
	log.Printf("Denied %s %s from %s: %s", r.Method, r.URL.Path, from, reason)
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// Restrict enforces both the origin requirement and the IP allowlist.
func (f *IPFilter) Restrict(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ip, reason, ok := f.allow(r, true); !ok {
			f.deny(w, r, ip, reason)
			return
		}
		handler(w, r)
	}
}

// RestrictOriginOnly enforces the origin requirement but skips the allowlist.
// Anonymous share links use this: they are token-gated with their own expiry
// and are meant to be opened by people outside the allowlist, but they should
// still not be reachable by going around the CDN.
func (f *IPFilter) RestrictOriginOnly(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ip, reason, ok := f.allow(r, false); !ok {
			f.deny(w, r, ip, reason)
			return
		}
		handler(w, r)
	}
}
