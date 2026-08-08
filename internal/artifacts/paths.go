package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Default directory permission for run/artifact trees (Unix; Windows ignores bits).
const DefaultDirPerm os.FileMode = 0o755

// Default file permission for artifact files.
const DefaultFilePerm os.FileMode = 0o644

// RunsRoot returns the runs directory root.
// Override with ASH_RUNS_DIR (absolute or relative); default is <dataDir>/runs.
func RunsRoot(dataDir string) string {
	if v := strings.TrimSpace(os.Getenv("ASH_RUNS_DIR")); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(dataDir, "runs")
}

// RunDir returns <runsRoot>/<runID>.
func RunDir(dataDir, runID string) string {
	return filepath.Join(RunsRoot(dataDir), runID)
}

// ObjectStoreRoot returns <dataDir>/object_store (fs artifact backend).
func ObjectStoreRoot(dataDir string) string {
	return filepath.Join(dataDir, "object_store")
}

// EnsureRunLayout creates runDir plus artifacts/checkpoints/audit with DefaultDirPerm.
func EnsureRunLayout(runDir string) error {
	for _, sub := range []string{"", "artifacts", "checkpoints", "audit"} {
		p := runDir
		if sub != "" {
			p = filepath.Join(runDir, sub)
		}
		if err := os.MkdirAll(p, DefaultDirPerm); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
	}
	return nil
}

// MaxArtifactsBytes returns ASH_ARTIFACTS_MAX_BYTES (0 = unlimited).
func MaxArtifactsBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("ASH_ARTIFACTS_MAX_BYTES"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// PathProfile describes effective artifact path strategy (appendix F §3).
type PathProfile struct {
	Platform        string `json:"platform"`
	DataDir         string `json:"dataDir"`
	RunsRoot        string `json:"runsRoot"`
	ObjectStoreRoot string `json:"objectStoreRoot"`
	RunsDirOverride bool   `json:"runsDirOverride"`
	DirPerm         string `json:"dirPerm"`
	FilePerm        string `json:"filePerm"`
	MaxBytes        int64  `json:"maxArtifactsBytes,omitempty"`
	PathSeparator   string `json:"pathSeparator"`
	Notes           string `json:"notes,omitempty"`
}

// DescribePaths returns the active path strategy for storage/profile and docs.
func DescribePaths(dataDir string) PathProfile {
	override := strings.TrimSpace(os.Getenv("ASH_RUNS_DIR")) != ""
	notes := "Unix dirs 0755 / files 0644; Windows uses ACLs inherited from parent. Artifact relative URIs use '/'."
	if runtime.GOOS == "windows" {
		notes = "Windows: MkdirAll inherits parent ACLs; prefer ASH_DATA_DIR under user profile or repo .ash; WSL should set ASH_DATA_DIR/ASH_RUNS_DIR inside the Linux filesystem for LF-native tools."
	}
	return PathProfile{
		Platform:        runtime.GOOS,
		DataDir:         dataDir,
		RunsRoot:        RunsRoot(dataDir),
		ObjectStoreRoot: ObjectStoreRoot(dataDir),
		RunsDirOverride: override,
		DirPerm:         "0755",
		FilePerm:        "0644",
		MaxBytes:        MaxArtifactsBytes(),
		PathSeparator:   string(os.PathSeparator),
		Notes:           notes,
	}
}
