package remote

import "github.com/ash-repwiki/ash/internal/sandbox"

func init() {
	sandbox.RegisterRemoteAvailable(Available)
}
