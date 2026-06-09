package model

// ComponentKind describes the syntactic kind of a smart contract artifact.
type ComponentKind string

const (
	KindContract  ComponentKind = "contract"
	KindInterface ComponentKind = "interface"
	KindLibrary   ComponentKind = "library"
	KindAbstract  ComponentKind = "abstract"
	KindExternal  ComponentKind = "external" // third-party / registry package
)

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

// Function represents a parsed Solidity function signature.
type Function struct {
	Name       string
	Visibility string // public, private, internal, external
	Mutability string // pure, view, payable, nonpayable
}

// Component is the canonical domain model for a blockchain contract or library.
// All downstream systems (graph, semantic, cyclonedx) consume this type.
type Component struct {
	ID          string
	Name        string
	Kind        ComponentKind
	Type        ComponentType
	SourceFile  string
	Imports     []string
	Inherits    []string
	Functions   []Function
	Events      []string
	Modifiers   []string
	IsExternal  bool
	PackageName string // e.g. "@openzeppelin/contracts"
	Version     string
	Tags        map[string]string // free-form key/value for semantic metadata
}

// NewComponent creates a Component with sensible defaults.
func NewComponent(name string, kind ComponentKind) *Component {
	return &Component{
		Name: name,
		Kind: kind,
		Type: ComponentTypeGeneric,
		Tags: make(map[string]string),
	}
}
