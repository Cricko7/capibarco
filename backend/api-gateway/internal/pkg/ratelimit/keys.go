// Package ratelimit contains traffic shaping helpers.
package ratelimit

import "github.com/petmatch/petmatch/internal/app/gateway"

// KeysForRequest returns all rate-limit buckets that apply to a request.
func KeysForRequest(ip string, principal gateway.Principal) []string {
	keys := make([]string, 0, 2+len(principal.Roles))
	if ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	if principal.ActorID != "" {
		keys = append(keys, "actor:"+principal.ActorID)
	}
	for _, role := range principal.Roles {
		if role != "" {
			keys = append(keys, "role:"+role)
		}
	}
	return keys
}
