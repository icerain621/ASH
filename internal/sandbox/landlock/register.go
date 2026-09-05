package landlock

import "github.com/ash-repwiki/ash/internal/sandbox"

func init() {
	sandbox.RegisterLandlockAvailable(Available)
}
