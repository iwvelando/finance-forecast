// Package optimization provides shared data structures for optimization results.
package optimization

// Summary captures the result of a single optimization directive.
type Summary struct {
	Scope           string   `json:"scope"`
	TargetName      string   `json:"targetName"`
	Field           string   `json:"field"`
	Original        float64  `json:"original"`
	Value           float64  `json:"value"`
	Floor           float64  `json:"floor"`
	MinimumCash     float64  `json:"minimumCash"`
	Headroom        float64  `json:"headroom"`
	Iterations      int      `json:"iterations"`
	Converged       bool     `json:"converged"`
	Notes           []string `json:"notes,omitempty"`
	OriginalDisplay string   `json:"originalDisplay,omitempty"`
	ValueDisplay    string   `json:"valueDisplay,omitempty"`
}
