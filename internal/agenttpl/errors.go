package agenttpl

import "errors"

// Sentinel errors for production binding (H05).
var (
	ErrNotApproved = errors.New("template is not approved for production")
	ErrUnknown     = errors.New("unknown template")
)
