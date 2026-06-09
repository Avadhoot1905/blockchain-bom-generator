package semantic

import "github.com/smartbom/smartbom/internal/graph"

// Analyzer classifies and annotates nodes in the dependency graph.
// Multiple analyzers are composed by the Pipeline type.
type Analyzer interface {
	// Analyze inspects the graph and updates node metadata in place.
	Analyze(g *graph.Graph) error

	// Name returns a human-readable identifier for logging.
	Name() string
}

// Pipeline runs a sequence of Analyzers against a graph.
type Pipeline struct {
	analyzers []Analyzer
}

// NewPipeline creates a Pipeline with the provided analyzers.
func NewPipeline(analyzers ...Analyzer) *Pipeline {
	return &Pipeline{analyzers: analyzers}
}

// Run executes all analyzers in order, stopping on the first error.
func (p *Pipeline) Run(g *graph.Graph) error {
	for _, a := range p.analyzers {
		if err := a.Analyze(g); err != nil {
			return err
		}
	}
	return nil
}

// DefaultPipeline returns a Pipeline with all built-in analyzers.
func DefaultPipeline() *Pipeline {
	return NewPipeline(
		&TokenAnalyzer{},
		&ProxyAnalyzer{},
		&OracleAnalyzer{},
		&GovernanceAnalyzer{},
		&TreasuryAnalyzer{},
	)
}
