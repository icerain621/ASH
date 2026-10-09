package passk

import "fmt"

// Config runs one Pass^k sample for a frozen check.
type Config struct {
	K     int
	Check func(i int) error
}

// Cluster groups failures by error message.
type Cluster struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// Result is the Pass^k report.
type Result struct {
	K            int       `json:"k"`
	Passes       int       `json:"passes"`
	Fails        int       `json:"fails"`
	AllPass      bool      `json:"allPass"`
	FailClusters []Cluster `json:"failClusters,omitempty"`
}

// Run executes Check k times. It must not be wired into Doctor startup.
func Run(cfg Config) (Result, error) {
	if cfg.K < 1 {
		return Result{}, fmt.Errorf("k must be >= 1")
	}
	if cfg.Check == nil {
		return Result{}, fmt.Errorf("check is required")
	}
	out := Result{K: cfg.K}
	clusters := map[string]int{}
	for i := 0; i < cfg.K; i++ {
		if err := cfg.Check(i); err != nil {
			out.Fails++
			clusters[err.Error()]++
			continue
		}
		out.Passes++
	}
	out.AllPass = out.Fails == 0
	for reason, n := range clusters {
		out.FailClusters = append(out.FailClusters, Cluster{Reason: reason, Count: n})
	}
	return out, nil
}
