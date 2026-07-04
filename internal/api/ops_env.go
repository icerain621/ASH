package api

import "github.com/ash-repwiki/ash/internal/opsenv"

func workerOpsSnapshot() opsenv.Snapshot {
	return opsenv.Load()
}
