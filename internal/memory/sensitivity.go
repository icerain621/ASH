package memory

import "github.com/ash-repwiki/ash/internal/store"

const ClearanceNormal = "normal"

func sensitivityRank(level string) int {
	switch level {
	case "", ClearanceNormal:
		return 0
	case "restricted":
		return 1
	case "secret":
		return 2
	default:
		return 3
	}
}

// Visible reports whether a record at recordSensitivity may be returned to clearance.
func Visible(recordSensitivity, clearance string) bool {
	if clearance == "" {
		clearance = ClearanceNormal
	}
	return sensitivityRank(recordSensitivity) <= sensitivityRank(clearance)
}

// FilterBySensitivity drops records the caller is not cleared to read.
func FilterBySensitivity(rows []store.MemoryRecord, clearance string) []store.MemoryRecord {
	if len(rows) == 0 {
		return nil
	}
	out := make([]store.MemoryRecord, 0, len(rows))
	for _, row := range rows {
		if Visible(row.Sensitivity, clearance) {
			out = append(out, row)
		}
	}
	return out
}
