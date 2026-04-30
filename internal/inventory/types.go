package inventory

import "time"

type Classification string

const (
	Portable     Classification = "portable"
	MachineLocal Classification = "machine-local"
	SecretAuth   Classification = "secret-auth"
	RuntimeCache Classification = "runtime-cache"
	AppOwned     Classification = "app-owned"
	Unknown      Classification = "unknown"
)

type RiskLevel string

const (
	RiskInfo     RiskLevel = "info"
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type FixKind string

const (
	FixPinPackage          FixKind = "pin-package"
	FixExternalizeSecret   FixKind = "externalize-secret"
	FixReplaceShellWrapper FixKind = "replace-shell-wrapper"
	FixNarrowFilesystem    FixKind = "narrow-filesystem"
	FixManualReview        FixKind = "manual-review"
	FixIgnoreWithReason    FixKind = "ignore-with-reason"
)

type Item struct {
	ID             string            `json:"id"`
	Tool           string            `json:"tool"`
	Path           string            `json:"path"`
	Kind           string            `json:"kind"`
	Classification Classification    `json:"classification"`
	Risk           RiskLevel         `json:"risk"`
	Reason         string            `json:"reason"`
	Recommendation string            `json:"recommended_action"`
	Exists         bool              `json:"exists"`
	SizeBytes      int64             `json:"size_bytes,omitempty"`
	ModTime        *time.Time        `json:"mod_time,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type Finding struct {
	ID             string    `json:"id"`
	Tool           string    `json:"tool"`
	Path           string    `json:"path"`
	Severity       RiskLevel `json:"severity"`
	Rule           string    `json:"rule"`
	Message        string    `json:"message"`
	Evidence       string    `json:"evidence,omitempty"`
	Recommendation string    `json:"recommended_action"`
	Impact         string    `json:"impact,omitempty"`
	Why            string    `json:"why_this_matters,omitempty"`
	DocsURL        string    `json:"docs_url,omitempty"`
	FixAvailable   bool      `json:"fix_available"`
	FixKind        FixKind   `json:"fix_kind,omitempty"`
	Confidence     string    `json:"confidence,omitempty"`
	Risk           RiskLevel `json:"risk,omitempty"`
	RequiresReview bool      `json:"requires_review"`
	FixSummary     string    `json:"fix_summary,omitempty"`
	FixSteps       []string  `json:"fix_steps,omitempty"`
}

type AdapterStatus struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Available   bool     `json:"available"`
	Checked     []string `json:"checked"`
	Found       []string `json:"found,omitempty"`
}

type Summary struct {
	TotalItems       int                    `json:"total_items"`
	TotalFindings    int                    `json:"total_findings"`
	ByClassification map[Classification]int `json:"by_classification"`
	ByRisk           map[RiskLevel]int      `json:"by_risk"`
	Tools            map[string]int         `json:"tools"`
}

type Report struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Hostname    string          `json:"hostname"`
	Home        string          `json:"home"`
	Summary     Summary         `json:"summary"`
	Items       []Item          `json:"items"`
	Findings    []Finding       `json:"findings"`
	Adapters    []AdapterStatus `json:"adapters"`
}
