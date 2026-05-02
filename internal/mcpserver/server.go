package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jsonbored/nightward/internal/analysis"
	"github.com/jsonbored/nightward/internal/fixplan"
	"github.com/jsonbored/nightward/internal/inventory"
	"github.com/jsonbored/nightward/internal/policy"
	"github.com/jsonbored/nightward/internal/reportdiff"
	"github.com/jsonbored/nightward/internal/rules"
	"github.com/jsonbored/nightward/internal/schedule"
)

const (
	protocolVersion = "2025-06-18"
	maxTextBytes    = 512 * 1024
)

type Server struct {
	Home    string
	Version string
	Now     func() time.Time
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolResult struct {
	Content           []textContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type tool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

type resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type compactPolicyReport struct {
	SchemaVersion    int                 `json:"schema_version"`
	GeneratedAt      time.Time           `json:"generated_at"`
	Strict           bool                `json:"strict"`
	Passed           bool                `json:"passed"`
	Threshold        inventory.RiskLevel `json:"threshold"`
	Summary          policy.Summary      `json:"summary"`
	Violations       []compactViolation  `json:"violations,omitempty"`
	SignalViolations []compactViolation  `json:"signal_violations,omitempty"`
	Truncated        bool                `json:"truncated,omitempty"`
}

type compactViolation struct {
	ID        string              `json:"id"`
	Rule      string              `json:"rule"`
	Severity  inventory.RiskLevel `json:"severity"`
	Tool      string              `json:"tool,omitempty"`
	Provider  string              `json:"provider,omitempty"`
	Path      string              `json:"path,omitempty"`
	Server    string              `json:"server,omitempty"`
	Message   string              `json:"message"`
	Evidence  string              `json:"evidence,omitempty"`
	Recommend string              `json:"recommended_action,omitempty"`
}

func Serve(home, version string, stdin io.Reader, stdout io.Writer) error {
	server := Server{Home: home, Version: version, Now: time.Now}
	return server.Serve(stdin, stdout)
}

func (s Server) Serve(stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	writer := bufio.NewWriter(stdout)
	defer writer.Flush()
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			writeResponse(writer, response{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &responseError{Code: -32700, Message: "parse error"},
			})
			if err := writer.Flush(); err != nil {
				return err
			}
			continue
		}
		resp, ok := s.Handle(req)
		if !ok {
			continue
		}
		writeResponse(writer, resp)
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s Server) Handle(req request) (response, bool) {
	if len(req.ID) == 0 {
		return response{}, false
	}
	id := req.ID
	switch req.Method {
	case "initialize":
		return response{JSONRPC: "2.0", ID: id, Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{"listChanged": false},
				"resources": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]string{
				"name":    "nightward",
				"title":   "Nightward",
				"version": s.version(),
			},
			"instructions": "Nightward MCP is read-only. It audits local AI agent state, MCP config, provider posture, policy status, and plan-only remediation without mutating config.",
		}}, true
	case "ping":
		return response{JSONRPC: "2.0", ID: id, Result: map[string]any{}}, true
	case "tools/list":
		return response{JSONRPC: "2.0", ID: id, Result: map[string]any{"tools": mcpTools()}}, true
	case "tools/call":
		result, err := s.callTool(req.Params)
		if err != nil {
			return response{JSONRPC: "2.0", ID: id, Result: errorToolResult(err)}, true
		}
		return response{JSONRPC: "2.0", ID: id, Result: result}, true
	case "resources/list":
		return response{JSONRPC: "2.0", ID: id, Result: map[string]any{"resources": mcpResources()}}, true
	case "resources/read":
		result, err := s.readResource(req.Params)
		if err != nil {
			return response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: -32602, Message: err.Error()}}, true
		}
		return response{JSONRPC: "2.0", ID: id, Result: result}, true
	default:
		return response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: -32601, Message: "method not found"}}, true
	}
}

func (s Server) callTool(raw json.RawMessage) (callToolResult, error) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return callToolResult{}, fmt.Errorf("invalid tool call params")
	}
	switch params.Name {
	case "nightward_scan":
		report := s.scan(stringArg(params.Arguments, "workspace"))
		return successToolResult(bounded("scan", report))
	case "nightward_doctor":
		return successToolResult(s.doctor())
	case "nightward_findings":
		report := s.scan(stringArg(params.Arguments, "workspace"))
		return successToolResult(filterFindings(report.Findings, params.Arguments))
	case "nightward_explain_finding":
		report := s.scan(stringArg(params.Arguments, "workspace"))
		id := stringArg(params.Arguments, "finding_id")
		if id == "" {
			return callToolResult{}, errors.New("finding_id is required")
		}
		finding, ok := fixplan.Find(report, id)
		if !ok {
			return successToolResult(map[string]any{"found": false, "finding_id": id})
		}
		return successToolResult(map[string]any{"found": true, "finding": finding})
	case "nightward_fix_plan":
		report := s.scan(stringArg(params.Arguments, "workspace"))
		selector := fixplan.Selector{
			FindingID: stringArg(params.Arguments, "finding_id"),
			Rule:      stringArg(params.Arguments, "rule"),
			All:       boolArg(params.Arguments, "all") || stringArg(params.Arguments, "finding_id") == "" && stringArg(params.Arguments, "rule") == "",
		}
		return successToolResult(fixplan.Build(report, selector))
	case "nightward_report_changes":
		diff, ok, err := s.latestReportDiff(stringArg(params.Arguments, "report_dir"))
		if err != nil {
			return callToolResult{}, err
		}
		if !ok {
			return successToolResult(map[string]any{"available": false, "message": "Need at least two saved reports."})
		}
		return successToolResult(map[string]any{"available": true, "diff": diff})
	case "nightward_policy_check":
		report := s.scan(stringArg(params.Arguments, "workspace"))
		var analysisReport analysis.Report
		includeAnalysis := boolArg(params.Arguments, "include_analysis")
		if includeAnalysis {
			analysisReport = analysis.Run(report, analysis.Options{
				Mode:      scanMode(report),
				Workspace: report.Workspace,
				With:      stringListArg(params.Arguments, "providers"),
				Online:    false,
			})
		}
		policyReport := policy.CheckWithOptions(report, policy.Options{
			Strict:          boolArg(params.Arguments, "strict"),
			IncludeAnalysis: includeAnalysis,
			Analysis:        analysisReport,
		})
		if boolArg(params.Arguments, "compact") {
			return successToolResult(compactPolicy(policyReport, 25))
		}
		return successToolResult(policyReport)
	default:
		return callToolResult{}, fmt.Errorf("unknown tool %q", params.Name)
	}
}

func (s Server) readResource(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.URI == "" {
		return nil, errors.New("uri is required")
	}
	var value any
	switch params.URI {
	case "nightward://rules":
		value = rules.List()
	case "nightward://providers":
		value = analysis.Providers()
	case "nightward://schedule":
		value = schedule.Status(s.Home)
	case "nightward://latest-report":
		records := schedule.ReportHistory(defaultReportDir(s.Home), 1)
		if len(records) == 0 {
			value = map[string]any{"available": false}
			break
		}
		report, err := readReport(records[0].Path)
		if err != nil {
			return nil, err
		}
		value = map[string]any{"available": true, "path": records[0].Path, "report": bounded("latest_report", report)}
	default:
		return nil, fmt.Errorf("unsupported resource URI %q", params.URI)
	}
	text, err := jsonText(value)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"contents": []map[string]string{{
			"uri":      params.URI,
			"mimeType": "application/json",
			"text":     text,
		}},
	}, nil
}

func mcpTools() []tool {
	readOnly := map[string]any{"readOnlyHint": true}
	return []tool{
		{Name: "nightward_scan", Title: "Scan Nightward Scope", Description: "Run a read-only Nightward scan for HOME or an optional workspace.", InputSchema: objectSchema(map[string]any{"workspace": stringSchema("Optional workspace/repository path to scan instead of HOME.")}), Annotations: readOnly},
		{Name: "nightward_doctor", Title: "Nightward Doctor", Description: "Return local Nightward status, adapters, schedule state, and provider posture.", InputSchema: objectSchema(nil), Annotations: readOnly},
		{Name: "nightward_findings", Title: "List Nightward Findings", Description: "List redacted findings with optional severity, tool, rule, and search filters.", InputSchema: objectSchema(map[string]any{"workspace": stringSchema("Optional workspace/repository path."), "severity": stringSchema("Optional severity filter."), "tool": stringSchema("Optional tool filter."), "rule": stringSchema("Optional rule filter."), "search": stringSchema("Optional case-insensitive search text.")}), Annotations: readOnly},
		{Name: "nightward_explain_finding", Title: "Explain Finding", Description: "Return a single Nightward finding by full ID or unique prefix.", InputSchema: requiredObjectSchema(map[string]any{"workspace": stringSchema("Optional workspace/repository path."), "finding_id": stringSchema("Finding ID or unique prefix.")}, []string{"finding_id"}), Annotations: readOnly},
		{Name: "nightward_fix_plan", Title: "Generate Fix Plan", Description: "Generate plan-only remediation output for all findings or a selected finding/rule.", InputSchema: objectSchema(map[string]any{"workspace": stringSchema("Optional workspace/repository path."), "finding_id": stringSchema("Optional finding ID or unique prefix."), "rule": stringSchema("Optional rule ID."), "all": map[string]any{"type": "boolean", "description": "Include all findings."}}), Annotations: readOnly},
		{Name: "nightward_report_changes", Title: "Compare Latest Reports", Description: "Compare the latest two saved Nightward reports and return added, removed, and changed findings.", InputSchema: objectSchema(map[string]any{"report_dir": stringSchema("Optional report directory. Defaults to Nightward state reports.")}), Annotations: readOnly},
		{Name: "nightward_policy_check", Title: "Check Policy", Description: "Run the local policy gate with optional offline analysis. Online providers are not enabled through MCP v1.", InputSchema: objectSchema(map[string]any{"workspace": stringSchema("Optional workspace/repository path."), "strict": map[string]any{"type": "boolean", "description": "Fail on medium or higher findings."}, "include_analysis": map[string]any{"type": "boolean", "description": "Include offline analysis signals."}, "compact": map[string]any{"type": "boolean", "description": "Return a compact AI-client friendly policy summary instead of full violations."}, "providers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Offline providers to request for analysis."}}), Annotations: readOnly},
	}
}

func mcpResources() []resource {
	return []resource{
		{URI: "nightward://rules", Name: "rules", Title: "Nightward Rules", Description: "Rule metadata and recommendations.", MimeType: "application/json"},
		{URI: "nightward://providers", Name: "providers", Title: "Nightward Providers", Description: "Provider support and privacy metadata.", MimeType: "application/json"},
		{URI: "nightward://schedule", Name: "schedule", Title: "Nightward Schedule", Description: "Local schedule/report-history status.", MimeType: "application/json"},
		{URI: "nightward://latest-report", Name: "latest-report", Title: "Latest Nightward Report", Description: "Latest saved report if one exists.", MimeType: "application/json"},
	}
}

func (s Server) scan(workspace string) inventory.Report {
	workspace = strings.TrimSpace(workspace)
	if workspace != "" {
		return inventory.NewWorkspaceScanner(expandHome(s.Home, workspace)).Scan()
	}
	return inventory.NewScanner(s.Home).Scan()
}

func (s Server) doctor() map[string]any {
	return map[string]any{
		"schema_version": 1,
		"generated_at":   s.now().UTC(),
		"version":        s.version(),
		"home":           s.Home,
		"schedule":       schedule.Status(s.Home),
		"adapters":       s.scan("").Adapters,
		"providers":      analysis.ProviderStatuses(nil, false),
	}
}

func (s Server) latestReportDiff(reportDir string) (reportdiff.Diff, bool, error) {
	if strings.TrimSpace(reportDir) == "" {
		reportDir = defaultReportDir(s.Home)
	} else {
		reportDir = expandHome(s.Home, reportDir)
	}
	history := schedule.ReportHistory(filepath.Clean(reportDir), 2)
	if len(history) < 2 {
		return reportdiff.Diff{}, false, nil
	}
	after, err := readReport(history[0].Path)
	if err != nil {
		return reportdiff.Diff{}, false, err
	}
	before, err := readReport(history[1].Path)
	if err != nil {
		return reportdiff.Diff{}, false, err
	}
	return reportdiff.Compare(history[1].Path, history[0].Path, before, after), true, nil
}

func successToolResult(value any) (callToolResult, error) {
	text, err := jsonText(value)
	if err != nil {
		return callToolResult{}, err
	}
	return callToolResult{
		Content:           []textContent{{Type: "text", Text: text}},
		StructuredContent: value,
	}, nil
}

func errorToolResult(err error) callToolResult {
	message := err.Error()
	if len(message) > 500 {
		message = message[:500] + "..."
	}
	return callToolResult{
		Content: []textContent{{Type: "text", Text: message}},
		IsError: true,
	}
}

func jsonText(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	if len(data) <= maxTextBytes {
		return string(data), nil
	}
	truncated := map[string]any{
		"truncated": true,
		"max_bytes": maxTextBytes,
		"message":   "Nightward MCP output exceeded the local safety cap. Narrow the request with filters.",
	}
	data, err = json.MarshalIndent(truncated, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func bounded(kind string, value any) any {
	data, err := json.Marshal(value)
	if err != nil || len(data) <= maxTextBytes {
		return value
	}
	switch report := value.(type) {
	case inventory.Report:
		summary := report
		if len(summary.Items) > 100 {
			summary.Items = summary.Items[:100]
		}
		if len(summary.Findings) > 100 {
			summary.Findings = summary.Findings[:100]
		}
		return map[string]any{
			"truncated": true,
			"kind":      kind,
			"summary":   summary.Summary,
			"items":     summary.Items,
			"findings":  summary.Findings,
		}
	default:
		return map[string]any{
			"truncated": true,
			"kind":      kind,
			"message":   "Output exceeded the Nightward MCP safety cap.",
		}
	}
}

func compactPolicy(report policy.Report, limit int) compactPolicyReport {
	if limit <= 0 {
		limit = 25
	}
	out := compactPolicyReport{
		SchemaVersion: report.SchemaVersion,
		GeneratedAt:   report.GeneratedAt,
		Strict:        report.Strict,
		Passed:        report.Passed,
		Threshold:     report.Threshold,
		Summary:       report.Summary,
	}
	for _, finding := range report.Violations {
		if len(out.Violations) >= limit {
			out.Truncated = true
			break
		}
		out.Violations = append(out.Violations, compactViolation{
			ID:        finding.ID,
			Rule:      finding.Rule,
			Severity:  finding.Severity,
			Tool:      finding.Tool,
			Path:      finding.Path,
			Server:    finding.Server,
			Message:   finding.Message,
			Evidence:  finding.Evidence,
			Recommend: finding.Recommendation,
		})
	}
	for _, signal := range report.SignalViolations {
		if len(out.SignalViolations) >= limit {
			out.Truncated = true
			break
		}
		out.SignalViolations = append(out.SignalViolations, compactViolation{
			ID:        signal.ID,
			Rule:      signal.Rule,
			Severity:  signal.Severity,
			Provider:  signal.Provider,
			Path:      signal.Path,
			Message:   signal.Message,
			Evidence:  signal.Evidence,
			Recommend: signal.Recommendation,
		})
	}
	if len(report.Violations) > len(out.Violations) || len(report.SignalViolations) > len(out.SignalViolations) {
		out.Truncated = true
	}
	return out
}

func filterFindings(findings []inventory.Finding, args map[string]any) []inventory.Finding {
	severity := strings.ToLower(stringArg(args, "severity"))
	tool := strings.ToLower(stringArg(args, "tool"))
	rule := strings.ToLower(stringArg(args, "rule"))
	search := strings.ToLower(stringArg(args, "search"))
	out := make([]inventory.Finding, 0, len(findings))
	for _, finding := range findings {
		if severity != "" && strings.ToLower(string(finding.Severity)) != severity {
			continue
		}
		if tool != "" && strings.ToLower(finding.Tool) != tool {
			continue
		}
		if rule != "" && strings.ToLower(finding.Rule) != rule {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(strings.Join([]string{finding.ID, finding.Tool, finding.Path, finding.Server, finding.Rule, finding.Message, finding.Evidence, finding.Recommendation}, "\n")), search) {
			continue
		}
		out = append(out, finding)
	}
	return out
}

func readReport(path string) (inventory.Report, error) {
	var report inventory.Report
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- MCP resource reads only Nightward report paths selected from local state or explicit user args.
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	if report.SchemaVersion == 0 {
		report.SchemaVersion = inventory.ReportSchemaVersion
	}
	return report, nil
}

func writeResponse(w io.Writer, resp response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = w.Write(append(data, '\n'))
}

func objectSchema(properties map[string]any) map[string]any {
	return requiredObjectSchema(properties, nil)
}

func requiredObjectSchema(properties map[string]any, required []string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}

func boolArg(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	value, _ := args[key].(bool)
	return value
}

func stringListArg(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	switch value := args[key].(type) {
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	case string:
		return splitCSV(value)
	default:
		return nil
	}
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func expandHome(home, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func defaultReportDir(home string) string {
	return filepath.Join(home, ".local", "state", "nightward", "reports")
}

func scanMode(report inventory.Report) string {
	if report.ScanMode != "" {
		return report.ScanMode
	}
	if report.Workspace != "" {
		return "workspace"
	}
	return "home"
}

func (s Server) version() string {
	if strings.TrimSpace(s.Version) == "" {
		return "devel"
	}
	return s.Version
}

func (s Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
