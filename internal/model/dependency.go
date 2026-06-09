package model

// RelationshipKind describes how one component depends on another.
type RelationshipKind string

const (
	RelImports  RelationshipKind = "imports"
	RelInherits RelationshipKind = "inherits"
	RelUses     RelationshipKind = "uses"
	RelDeploys  RelationshipKind = "deploys"
)

// Dependency captures a directed relationship between two components.
type Dependency struct {
	SourceID     string
	TargetID     string
	Relationship RelationshipKind
}
