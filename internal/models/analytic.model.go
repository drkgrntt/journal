package models

import "time"

// RequestLogRetention is how long request log rows are kept.
const RequestLogRetention = 14 * 24 * time.Hour

func init() {
	registerModel(&Analytic{})
}

// Analytic is one row of the request log. It is deliberately anonymous: no IP
// address, user agent, query string, or account ID (CreatorID is always
// uuid.Nil). It exists to spot errors and outages, and rows are deleted after
// RequestLogRetention by the jobs package.
type Analytic struct {
	*Base
	Method string `json:"method,omitempty"`
	Domain string `json:"domain,omitempty"`
	Page   string `json:"page,omitempty"`
	Status int    `json:"status,omitempty"`

	// RequestID is a random per-request reference, not linked to any person or
	// to other requests. It is shown to the person on a server error so they can
	// quote it.
	RequestID string `json:"requestId,omitempty"`
}
