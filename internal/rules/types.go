package rules

const Version = "ash.rules/v0.1"

// Document is the top-level Rules/Scenario DSL document.
type Document struct {
	Version  string   `yaml:"version" json:"version"`
	Scenario Scenario `yaml:"scenario" json:"scenario"`
	Hooks    []Hook   `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

type Scenario struct {
	Name            string            `yaml:"name" json:"name"`
	ScenarioVersion string            `yaml:"scenarioVersion" json:"scenarioVersion"`
	Description     string            `yaml:"description,omitempty" json:"description,omitempty"`
	PolicyProfile   string            `yaml:"policyProfile,omitempty" json:"policyProfile,omitempty"`
	Skills          []string          `yaml:"skills,omitempty" json:"skills,omitempty"`
	Sandbox         *ScenarioSandbox  `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`
	Checkpoint      *CheckpointConfig `yaml:"checkpoint,omitempty" json:"checkpoint,omitempty"`
	Roles           map[string]Role   `yaml:"roles,omitempty" json:"roles,omitempty"`
	Inputs          *InputsSpec       `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Artifacts       *ArtifactsSpec    `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Gates           []Gate            `yaml:"gates,omitempty" json:"gates,omitempty"`
	Steps           []Step            `yaml:"steps" json:"steps"`
}

// ScenarioSandbox is the optional scenario-level sandbox floor (DX2).
type ScenarioSandbox struct {
	MinMode string `yaml:"minMode,omitempty" json:"minMode,omitempty"`
}

type CheckpointConfig struct {
	Strategy string `yaml:"strategy" json:"strategy"`
	Retain   int    `yaml:"retain" json:"retain"`
}

type Role struct {
	MaxParallel int `yaml:"maxParallel" json:"maxParallel"`
}

type InputsSpec struct {
	Required []string `yaml:"required,omitempty" json:"required,omitempty"`
	Optional []string `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type ArtifactsSpec struct {
	Required []ArtifactRef `yaml:"required,omitempty" json:"required,omitempty"`
}

type ArtifactRef struct {
	Type string `yaml:"type" json:"type"`
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

type Gate struct {
	ID       string        `yaml:"id" json:"id"`
	When     string        `yaml:"when" json:"when"`
	Blocking bool          `yaml:"blocking" json:"blocking"`
	Check    GateCheck     `yaml:"check" json:"check"`
	OnFail   *GateOnFail   `yaml:"onFail,omitempty" json:"onFail,omitempty"`
	Approval *ApprovalSpec `yaml:"approval,omitempty" json:"approval,omitempty"`
}

type GateCheck struct {
	Tool     string         `yaml:"tool,omitempty" json:"tool,omitempty"`
	Expect   map[string]any `yaml:"expect,omitempty" json:"expect,omitempty"`
	Artifact string         `yaml:"artifact,omitempty" json:"artifact,omitempty"`
}

type GateOnFail struct {
	Message     string          `yaml:"message" json:"message"`
	Remediation []ToolChainItem `yaml:"remediation,omitempty" json:"remediation,omitempty"`
}

type Step struct {
	ID        string          `yaml:"id" json:"id"`
	Role      string          `yaml:"role" json:"role"`
	Kind      string          `yaml:"kind" json:"kind"`
	PromptRef string          `yaml:"promptRef,omitempty" json:"promptRef,omitempty"`
	RAG       *RAGSpec        `yaml:"rag,omitempty" json:"rag,omitempty"`
	Agent     *AgentSpec      `yaml:"agent,omitempty" json:"agent,omitempty"`
	Chain     []ToolChainItem `yaml:"chain,omitempty" json:"chain,omitempty"`
	Verify    *VerifySpec     `yaml:"verify,omitempty" json:"verify,omitempty"`
	Gates     []string        `yaml:"gates,omitempty" json:"gates,omitempty"`
	Outputs   *StepOutputs    `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	TimeoutMs int64           `yaml:"timeoutMs,omitempty" json:"timeoutMs,omitempty"`
	Retry     *RetrySpec      `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// VerifySpec defines kind:verify checks (test/lint/status style tool calls).
type VerifySpec struct {
	Checks []ToolChainItem `yaml:"checks" json:"checks"`
	OnFail string          `yaml:"onFail,omitempty" json:"onFail,omitempty"` // fail|improve
}

type RAGSpec struct {
	Sources            []string `yaml:"sources" json:"sources"`
	RequireCitations   bool     `yaml:"requireCitations" json:"requireCitations"`
	OnMissingCitations string   `yaml:"onMissingCitations,omitempty" json:"onMissingCitations,omitempty"` // block|human_confirm
}

type ToolChainItem struct {
	Tool      string         `yaml:"tool" json:"tool"`
	Args      map[string]any `yaml:"args,omitempty" json:"args,omitempty"`
	TimeoutMs int64          `yaml:"timeoutMs,omitempty" json:"timeoutMs,omitempty"`
	Retry     *RetrySpec     `yaml:"retry,omitempty" json:"retry,omitempty"`
	Policy    string         `yaml:"policy,omitempty" json:"policy,omitempty"`
}

type AgentSpec struct {
	Adapter      string   `yaml:"adapter,omitempty" json:"adapter,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Prompt       string   `yaml:"prompt,omitempty" json:"prompt,omitempty"`
}

type RetrySpec struct {
	MaxAttempts int `yaml:"maxAttempts,omitempty" json:"maxAttempts,omitempty"`
	BackoffMs   int `yaml:"backoffMs,omitempty" json:"backoffMs,omitempty"`
}

type ApprovalSpec struct {
	Required bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Roles    []string `yaml:"roles,omitempty" json:"roles,omitempty"`
	Reason   string   `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type StepOutputs struct {
	Artifacts        []ArtifactRef     `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	MemoryCandidates []MemoryCandidate `yaml:"memoryCandidates,omitempty" json:"memoryCandidates,omitempty"`
}

type MemoryCandidate struct {
	Layer        string   `yaml:"layer" json:"layer"`
	Title        string   `yaml:"title,omitempty" json:"title,omitempty"`
	EvidenceFrom []string `yaml:"evidenceFrom,omitempty" json:"evidenceFrom,omitempty"`
}

type Hook struct {
	ID     string     `yaml:"id" json:"id"`
	On     string     `yaml:"on" json:"on"`
	Policy string     `yaml:"policy" json:"policy"`
	Rules  []HookRule `yaml:"rules" json:"rules"`
}

type HookRule struct {
	Match  map[string]any `yaml:"match" json:"match"`
	Action HookAction     `yaml:"action" json:"action"`
}

type HookAction struct {
	Deny   bool   `yaml:"deny" json:"deny"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// ValidationIssue describes a DSL validation error with location.
type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationResult struct {
	OK     bool              `json:"ok"`
	Issues []ValidationIssue `json:"issues,omitempty"`
	Doc    *Document         `json:"-"`
}

func (s Scenario) Ref() ScenarioRef {
	return ScenarioRef{Name: s.Name, ScenarioVersion: s.ScenarioVersion}
}

type ScenarioRef struct {
	Name            string `json:"name"`
	ScenarioVersion string `json:"scenarioVersion"`
}

type ScenarioSummary struct {
	Name            string `json:"name"`
	ScenarioVersion string `json:"scenarioVersion"`
	Description     string `json:"description,omitempty"`
	PolicyProfile   string `json:"policyProfile,omitempty"`
	StepCount       int    `json:"stepCount"`
	GateCount       int    `json:"gateCount"`
}
