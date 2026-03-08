package types

type Config struct {
	Delays      Delays       `json:"delays"`
	Alterations []Alteration `json:"alterations"`
}

type Delays struct {
	GlobalDelayMs *int           `json:"global_delay_ms,omitempty"`
	Patterns      []PatternDelay `json:"patterns"`
}

type PatternDelay struct {
	Pattern string `json:"pattern"`
	DelayMs int    `json:"delay_ms"`
}

type Alteration struct {
	URLPattern string `json:"url_pattern"`
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}
