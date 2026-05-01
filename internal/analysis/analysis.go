package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/inventory"
)

type SubjectType string

const (
	SubjectFinding SubjectType = "finding"
	SubjectItem    SubjectType = "item"
	SubjectPackage SubjectType = "package"
)

type SignalCategory string

const (
	CategorySupplyChain SignalCategory = "supply-chain"
	CategorySecrets     SignalCategory = "secrets-exposure"
	CategoryFilesystem  SignalCategory = "filesystem-scope"
	CategoryNetwork     SignalCategory = "network-exposure"
	CategoryExecution   SignalCategory = "execution-risk"
	CategoryLocality    SignalCategory = "machine-locality"
	CategoryAppState    SignalCategory = "app-state"
	CategoryUnknown     SignalCategory = "unknown"
)

type Options struct {
	Mode      string
	Workspace string
	With      []string
	Online    bool
	Package   string
	FindingID string
}

type Report struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Mode        string           `json:"mode"`
	Workspace   string           `json:"workspace,omitempty"`
	Summary     Summary          `json:"summary"`
	Providers   []ProviderStatus `json:"providers"`
	Subjects    []Subject        `json:"subjects"`
	Signals     []Signal         `json:"signals"`
}

type Summary struct {
	TotalSubjects      int                         `json:"total_subjects"`
	TotalSignals       int                         `json:"total_signals"`
	SignalsBySeverity  map[inventory.RiskLevel]int `json:"signals_by_severity"`
	SignalsByCategory  map[SignalCategory]int      `json:"signals_by_category"`
	SignalsByProvider  map[string]int              `json:"signals_by_provider"`
	HighestSeverity    inventory.RiskLevel         `json:"highest_severity"`
	ProviderWarnings   int                         `json:"provider_warnings"`
	NoKnownRiskSignals bool                        `json:"no_known_risk_signals"`
}

type Subject struct {
	ID       string      `json:"id"`
	Type     SubjectType `json:"type"`
	Name     string      `json:"name"`
	Tool     string      `json:"tool,omitempty"`
	Path     string      `json:"path,omitempty"`
	Rule     string      `json:"rule,omitempty"`
	Package  string      `json:"package,omitempty"`
	Evidence string      `json:"evidence,omitempty"`
}

type Signal struct {
	ID             string              `json:"id"`
	Provider       string              `json:"provider"`
	Rule           string              `json:"rule"`
	Category       SignalCategory      `json:"category"`
	SubjectID      string              `json:"subject_id"`
	SubjectType    SubjectType         `json:"subject_type"`
	Path           string              `json:"path,omitempty"`
	Severity       inventory.RiskLevel `json:"severity"`
	Confidence     string              `json:"confidence"`
	Message        string              `json:"message"`
	Evidence       string              `json:"evidence,omitempty"`
	Recommendation string              `json:"recommended_action"`
	Why            string              `json:"why_this_matters,omitempty"`
}

type Provider struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Command      string `json:"command,omitempty"`
	Online       bool   `json:"online"`
	Default      bool   `json:"default"`
	Privacy      string `json:"privacy"`
	Capabilities string `json:"capabilities"`
}

type ProviderStatus struct {
	Provider
	Enabled   bool   `json:"enabled"`
	Available bool   `json:"available"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}

func Run(report inventory.Report, options Options) Report {
	if options.Mode == "" {
		options.Mode = "home"
	}
	out := Report{
		GeneratedAt: report.GeneratedAt,
		Mode:        options.Mode,
		Workspace:   options.Workspace,
		Summary: Summary{
			SignalsBySeverity: map[inventory.RiskLevel]int{},
			SignalsByCategory: map[SignalCategory]int{},
			SignalsByProvider: map[string]int{},
		},
		Providers: ProviderStatuses(options.With, options.Online),
	}

	if options.Package != "" {
		subject := Subject{
			ID:      stableID("package", options.Package),
			Type:    SubjectPackage,
			Name:    options.Package,
			Package: options.Package,
		}
		out.Subjects = append(out.Subjects, subject)
		out.Signals = append(out.Signals, Signal{
			ID:             stableID("signal", "nightward", "package_review", options.Package),
			Provider:       "nightward",
			Rule:           "package_review",
			Category:       CategorySupplyChain,
			SubjectID:      subject.ID,
			SubjectType:    subject.Type,
			Severity:       inventory.RiskInfo,
			Confidence:     "low",
			Message:        "Package analysis is structural only without an explicit provider.",
			Evidence:       "package=" + options.Package,
			Recommendation: "Run with an explicit provider after reviewing provider privacy behavior.",
			Why:            "Nightward avoids making safety claims about packages without registry or vulnerability evidence.",
		})
	}

	for _, finding := range report.Findings {
		if options.FindingID != "" && finding.ID != options.FindingID && !strings.HasPrefix(finding.ID, options.FindingID) {
			continue
		}
		subject := Subject{
			ID:       stableID("finding", finding.ID),
			Type:     SubjectFinding,
			Name:     finding.ID,
			Tool:     finding.Tool,
			Path:     finding.Path,
			Rule:     finding.Rule,
			Evidence: finding.Evidence,
		}
		out.Subjects = append(out.Subjects, subject)
		out.Signals = append(out.Signals, signalFromFinding(subject, finding))
	}

	if options.FindingID == "" && options.Package == "" {
		for _, item := range report.Items {
			if signal, ok := signalFromItem(item); ok {
				subject := Subject{
					ID:       stableID("item", item.ID),
					Type:     SubjectItem,
					Name:     item.ID,
					Tool:     item.Tool,
					Path:     item.Path,
					Evidence: string(item.Classification),
				}
				out.Subjects = append(out.Subjects, subject)
				signal.SubjectID = subject.ID
				out.Signals = append(out.Signals, signal)
			}
		}
	}
	if options.FindingID == "" && options.Package == "" {
		appendProviderSignals(&out, report, options)
	}

	sort.Slice(out.Subjects, func(i, j int) bool {
		if out.Subjects[i].Type == out.Subjects[j].Type {
			return out.Subjects[i].Name < out.Subjects[j].Name
		}
		return out.Subjects[i].Type < out.Subjects[j].Type
	})
	sort.Slice(out.Signals, func(i, j int) bool {
		if out.Signals[i].Severity == out.Signals[j].Severity {
			return out.Signals[i].ID < out.Signals[j].ID
		}
		return riskRank(out.Signals[i].Severity) > riskRank(out.Signals[j].Severity)
	})
	finalize(&out)
	return out
}

func Explain(report Report, idOrPrefix string) (Signal, bool) {
	for _, signal := range report.Signals {
		if signal.ID == idOrPrefix {
			return signal, true
		}
	}
	var matched []Signal
	for _, signal := range report.Signals {
		if strings.HasPrefix(signal.ID, idOrPrefix) {
			matched = append(matched, signal)
		}
	}
	if len(matched) == 1 {
		return matched[0], true
	}
	return Signal{}, false
}

func Providers() []Provider {
	return []Provider{
		{Name: "nightward", Kind: "built-in", Default: true, Privacy: "offline; reads only the selected HOME or workspace", Capabilities: "MCP, dotfiles, secret-path, filesystem, and local-endpoint heuristics"},
		{Name: "gitleaks", Kind: "local-command", Command: "gitleaks", Privacy: "local command; scans selected files when explicitly run", Capabilities: "secret pattern scanning"},
		{Name: "trufflehog", Kind: "local-command", Command: "trufflehog", Privacy: "local command; scans selected files when explicitly run", Capabilities: "secret pattern scanning with verification disabled by default"},
		{Name: "semgrep", Kind: "local-command", Command: "semgrep", Privacy: "local command; rule packs may require network if user config chooses that", Capabilities: "static analysis and malicious dependency rules"},
		{Name: "trivy", Kind: "local-command", Command: "trivy", Online: true, Privacy: "network-capable; vulnerability database updates may contact upstream services", Capabilities: "filesystem, dependency, IaC, and secret scanning"},
		{Name: "osv-scanner", Kind: "local-command", Command: "osv-scanner", Online: true, Privacy: "network-capable; queries vulnerability data for dependency manifests", Capabilities: "open source vulnerability matching"},
		{Name: "socket", Kind: "local-command", Command: "socket", Online: true, Privacy: "network-capable; uploads dependency manifest metadata and creates a remote Socket scan artifact", Capabilities: "remote supply-chain scan creation and malicious package signals"},
	}
}

func ProviderStatuses(with []string, online bool) []ProviderStatus {
	selected := selectedProviders(with)
	out := make([]ProviderStatus, 0, len(Providers()))
	known := map[string]bool{}
	for _, provider := range Providers() {
		known[provider.Name] = true
		enabled := provider.Default || selected["all"] || selected[provider.Name]
		status := ProviderStatus{Provider: provider, Enabled: enabled}
		if provider.Kind == "built-in" {
			status.Available = true
			status.Status = "ready"
			status.Detail = "built-in offline heuristics are enabled by default"
			out = append(out, status)
			continue
		}
		if provider.Command != "" {
			if path, err := exec.LookPath(provider.Command); err == nil {
				status.Available = true
				status.Detail = path
			} else {
				status.Detail = "command not found on PATH"
			}
		}
		switch {
		case !enabled:
			status.Status = "available"
			if !status.Available {
				status.Status = "missing"
			}
		case provider.Online && !online:
			status.Status = "blocked"
			status.Detail = strings.TrimSpace(status.Detail + "; requires --online before Nightward will use it")
		case !status.Available:
			status.Status = "missing"
		default:
			status.Status = "ready"
			status.Detail = strings.TrimSpace(status.Detail + "; available for explicit provider runs")
		}
		out = append(out, status)
	}
	if !selected["all"] {
		var unknown []string
		for name := range selected {
			if !known[name] {
				unknown = append(unknown, name)
			}
		}
		sort.Strings(unknown)
		for _, name := range unknown {
			out = append(out, ProviderStatus{
				Provider: Provider{
					Name:    name,
					Kind:    "unknown",
					Privacy: "unknown provider; Nightward did not run anything",
				},
				Enabled: true,
				Status:  "unknown",
				Detail:  "provider is not recognized by this Nightward version",
			})
		}
	}
	return out
}

func signalFromFinding(subject Subject, finding inventory.Finding) Signal {
	category := categoryForRule(finding.Rule)
	confidence := finding.Confidence
	if confidence == "" {
		confidence = "medium"
	}
	return Signal{
		ID:             stableID("signal", "nightward", finding.Rule, finding.ID),
		Provider:       "nightward",
		Rule:           "nightward/" + finding.Rule,
		Category:       category,
		SubjectID:      subject.ID,
		SubjectType:    subject.Type,
		Path:           finding.Path,
		Severity:       finding.Severity,
		Confidence:     confidence,
		Message:        finding.Message,
		Evidence:       finding.Evidence,
		Recommendation: finding.Recommendation,
		Why:            finding.Why,
	}
}

func signalFromItem(item inventory.Item) (Signal, bool) {
	switch item.Classification {
	case inventory.SecretAuth:
		return Signal{
			ID:             stableID("signal", "nightward", "secret_auth_path", item.ID),
			Provider:       "nightward",
			Rule:           "nightward/secret_auth_path",
			Category:       CategorySecrets,
			SubjectType:    SubjectItem,
			Path:           item.Path,
			Severity:       inventory.RiskCritical,
			Confidence:     "high",
			Message:        "Secret or auth path is present in the scan scope.",
			Evidence:       "classification=secret-auth path=" + item.Path,
			Recommendation: "Exclude this path from portable dotfiles and backups.",
			Why:            "Credential-bearing files should remain in a password manager, keychain, or machine-local store.",
		}, true
	case inventory.MachineLocal:
		return Signal{
			ID:             stableID("signal", "nightward", "machine_local_path", item.ID),
			Provider:       "nightward",
			Rule:           "nightward/machine_local_path",
			Category:       CategoryLocality,
			SubjectType:    SubjectItem,
			Path:           item.Path,
			Severity:       inventory.RiskMedium,
			Confidence:     "medium",
			Message:        "Machine-local path is present in the scan scope.",
			Evidence:       "classification=machine-local path=" + item.Path,
			Recommendation: "Move machine-specific values to a local overlay or document them as setup prerequisites.",
			Why:            "Portable repos should not silently depend on one machine's paths, sockets, identities, or services.",
		}, true
	case inventory.AppOwned:
		return Signal{
			ID:             stableID("signal", "nightward", "app_owned_state", item.ID),
			Provider:       "nightward",
			Rule:           "nightward/app_owned_state",
			Category:       CategoryAppState,
			SubjectType:    SubjectItem,
			Path:           item.Path,
			Severity:       inventory.RiskLow,
			Confidence:     "medium",
			Message:        "App-owned runtime state is present in the scan scope.",
			Evidence:       "classification=app-owned path=" + item.Path,
			Recommendation: "Prefer app-supported export formats over syncing raw runtime state.",
			Why:            "Raw app databases and caches often contain local identifiers, private history, or binary state.",
		}, true
	default:
		return Signal{}, false
	}
}

func categoryForRule(rule string) SignalCategory {
	switch rule {
	case "mcp_unpinned_package":
		return CategorySupplyChain
	case "mcp_secret_env", "mcp_secret_header", "mcp_local_token_path":
		return CategorySecrets
	case "mcp_broad_filesystem":
		return CategoryFilesystem
	case "mcp_local_endpoint":
		return CategoryNetwork
	case "mcp_shell_command", "mcp_unknown_command":
		return CategoryExecution
	case "mcp_server_review":
		return CategoryExecution
	default:
		return CategoryUnknown
	}
}

func finalize(report *Report) {
	report.Summary.TotalSubjects = len(report.Subjects)
	report.Summary.TotalSignals = len(report.Signals)
	report.Summary.HighestSeverity = inventory.RiskInfo
	for _, signal := range report.Signals {
		report.Summary.SignalsBySeverity[signal.Severity]++
		report.Summary.SignalsByCategory[signal.Category]++
		report.Summary.SignalsByProvider[signal.Provider]++
		if riskRank(signal.Severity) > riskRank(report.Summary.HighestSeverity) {
			report.Summary.HighestSeverity = signal.Severity
		}
	}
	for _, provider := range report.Providers {
		if provider.Enabled && (provider.Status == "missing" || provider.Status == "blocked" || provider.Status == "unknown") {
			report.Summary.ProviderWarnings++
		}
	}
	report.Summary.NoKnownRiskSignals = len(report.Signals) == 0
}

func selectedProviders(with []string) map[string]bool {
	out := map[string]bool{}
	for _, raw := range with {
		for _, name := range strings.Split(raw, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if name != "" {
				out[name] = true
			}
		}
	}
	return out
}

func stableID(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])[:12]
}

func riskRank(risk inventory.RiskLevel) int {
	switch risk {
	case inventory.RiskCritical:
		return 5
	case inventory.RiskHigh:
		return 4
	case inventory.RiskMedium:
		return 3
	case inventory.RiskLow:
		return 2
	default:
		return 1
	}
}

func RelativeWorkspacePath(workspace, path string) string {
	if workspace == "" || path == "" {
		return path
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

func HumanProviderSummary(status ProviderStatus) string {
	availability := "missing"
	if status.Available {
		availability = "available"
	}
	if status.Enabled {
		return fmt.Sprintf("%s enabled %s (%s)", status.Name, status.Status, availability)
	}
	return fmt.Sprintf("%s disabled %s", status.Name, availability)
}
