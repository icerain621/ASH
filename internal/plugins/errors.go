package plugins

import "errors"

// Sentinel errors for production binding (D08 / P16).
var (
	ErrNotApproved = errors.New("component is not approved for production")
	ErrUnknown     = errors.New("unknown component")
	ErrInvalidBind = errors.New("invalid component production bind")
)
