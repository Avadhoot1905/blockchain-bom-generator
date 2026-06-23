// Package model provides shared semantic constants used by the analyzer pipeline.
//
// Architectural decision (Phase 1, 2026-06-22):
// The original package also contained Component and Dependency struct types that
// were never instantiated in the runtime pipeline — the graph package's Node/Edge
// types are the canonical runtime entities.  Those orphaned structs have been
// removed to eliminate confusion about the authoritative data model.
// The constants below are kept because all five semantic analyzers import and use
// them to populate node.Metadata values.
package model

// ComponentType is the semantic classification assigned by analyzers.
type ComponentType string

const (
	ComponentTypeGeneric    ComponentType = "Generic"
	ComponentTypeProxy      ComponentType = "Proxy"
	ComponentTypeOracle     ComponentType = "Oracle"
	ComponentTypeGovernance ComponentType = "Governance"
	ComponentTypeTreasury   ComponentType = "Treasury"
	ComponentTypeToken      ComponentType = "Token"
)

// TokenStandard captures the detected ERC standard.
type TokenStandard string

const (
	TokenStandardERC20   TokenStandard = "ERC20"
	TokenStandardERC721  TokenStandard = "ERC721"
	TokenStandardERC1155 TokenStandard = "ERC1155"
	TokenStandardUnknown TokenStandard = ""
)
