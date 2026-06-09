// Package vuln provides the extensible vulnerability scanning framework.
// Only interfaces and wiring are implemented here; concrete scanners will be
// added in future iterations.
package vuln

import "github.com/smartbom/smartbom/internal/graph"

// Severity classifies the risk level of a finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Finding represents a single vulnerability or security concern.
type Finding struct {
	ID          string
	Severity    Severity
	Component   string // node ID in the graph
	Title       string
	Description string
	Remediation string
	References  []string
}

// Scanner is the interface all vulnerability scanners must implement.
// Scanners receive the dependency graph and return zero or more findings.
type Scanner interface {
	// Scan analyzes the graph and returns any detected findings.
	Scan(g *graph.Graph) ([]Finding, error)

	// Name returns the scanner's identifier (used in reports and logs).
	Name() string
}

// Engine runs a set of Scanners and aggregates their findings.
type Engine struct {
	scanners []Scanner
}

// NewEngine creates an Engine with the provided scanners.
func NewEngine(scanners ...Scanner) *Engine {
	return &Engine{scanners: scanners}
}

// Run executes all scanners and returns the combined list of findings.
// Execution continues even if an individual scanner returns an error;
// errors are accumulated and returned after all scanners complete.
func (e *Engine) Run(g *graph.Graph) ([]Finding, []error) {
	var findings []Finding
	var errs []error

	for _, s := range e.scanners {
		f, err := s.Scan(g)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		findings = append(findings, f...)
	}
	return findings, errs
}

// --- placeholder scanners (not yet implemented) ---

// DependencyScanner will check component versions against vulnerability DBs.
type DependencyScanner struct{}

func (s *DependencyScanner) Name() string { return "dependency-vulnerability" }
func (s *DependencyScanner) Scan(_ *graph.Graph) ([]Finding, error) {
	return nil, nil // TODO: integrate with OSV / Snyk
}

// ReentrancyScanner will detect reentrancy patterns.
type ReentrancyScanner struct{}

func (s *ReentrancyScanner) Name() string { return "reentrancy" }
func (s *ReentrancyScanner) Scan(_ *graph.Graph) ([]Finding, error) {
	return nil, nil // TODO: analyze call graph for reentrancy
}

// DelegatecallScanner will flag unsafe delegatecall usage.
type DelegatecallScanner struct{}

func (s *DelegatecallScanner) Name() string { return "unsafe-delegatecall" }
func (s *DelegatecallScanner) Scan(_ *graph.Graph) ([]Finding, error) {
	return nil, nil // TODO: inspect contract bodies for delegatecall
}

// AccessControlScanner will detect missing access control on sensitive functions.
type AccessControlScanner struct{}

func (s *AccessControlScanner) Name() string { return "access-control" }
func (s *AccessControlScanner) Scan(_ *graph.Graph) ([]Finding, error) {
	return nil, nil // TODO: cross-reference function visibility with modifiers
}

// UpgradeabilityScanner will audit upgrade patterns for safety issues.
type UpgradeabilityScanner struct{}

func (s *UpgradeabilityScanner) Name() string { return "upgradeability" }
func (s *UpgradeabilityScanner) Scan(_ *graph.Graph) ([]Finding, error) {
	return nil, nil // TODO: check storage layout, initializer guards, etc.
}

// DefaultEngine returns an Engine pre-configured with all built-in scanners.
// All scanners are currently stubs; they will be activated as implemented.
func DefaultEngine() *Engine {
	return NewEngine(
		&DependencyScanner{},
		&ReentrancyScanner{},
		&DelegatecallScanner{},
		&AccessControlScanner{},
		&UpgradeabilityScanner{},
	)
}
