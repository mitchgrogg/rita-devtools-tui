package types

type Config struct {
	Delays      Delays      `json:"delays"`
	Alterations Alterations `json:"alterations"`
}

type Delays struct {
	GlobalDelayMs *int           `json:"global_ms,omitempty"`
	Patterns      []PatternDelay `json:"patterns"`
}

type PatternDelay struct {
	Pattern string `json:"pattern"`
	DelayMs int    `json:"delay_ms"`
}

// AlterationKind selects which side of a flow a rule applies to. It is also
// the last path segment of the /api/alterations/<kind> endpoints.
type AlterationKind string

const (
	AlterationRequest  AlterationKind = "request"
	AlterationResponse AlterationKind = "response"
)

type Alterations struct {
	Request  []Alteration `json:"request"`
	Response []Alteration `json:"response"`
}

// Alteration covers both kinds; the kind-specific fields are omitted when
// unset so an unused field never overwrites anything on the proxy.
type Alteration struct {
	URLPattern string `json:"url_pattern"`
	StatusCode int    `json:"status_code,omitempty"` // response only
	RewriteURL string `json:"rewrite_url,omitempty"` // request only
	Body       string `json:"body,omitempty"`
}
