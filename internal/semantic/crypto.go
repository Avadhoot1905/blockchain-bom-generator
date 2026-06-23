package semantic

import (
	"strings"

	"github.com/smartbom/smartbom/internal/graph"
)

// CryptoAnalyzer detects cryptographic primitive usage from imports, inheritance
// chains, contract names, and function names.  No AST or function-body analysis
// is performed; detection is limited to structural signals available from the
// regex-based Solidity parser.
//
// Detected primitives are written to:
//
//	node.Metadata["CryptoPrimitives"]  []string – individual algorithm names
//	node.Metadata["CryptoCategories"]  []string – high-level categories
type CryptoAnalyzer struct{}

func (a *CryptoAnalyzer) Name() string { return "crypto" }

// detection rule for a single cryptographic primitive.
type cryptoRule struct {
	primitive string // e.g. "ECDSA"
	category  string // e.g. "DigitalSignature"

	importContains  []string // case-insensitive substrings in import paths
	inheritsContains []string // case-insensitive substrings in base names
	nameContains    []string // case-insensitive substrings in the contract name
	funcContains    []string // case-insensitive substrings in function names
}

var cryptoRules = []cryptoRule{
	{
		primitive:       "ECDSA",
		category:        "DigitalSignature",
		importContains:  []string{"ecdsa"},
		inheritsContains: []string{"abstractsigner"},
		nameContains:    []string{"signer", "signature"},
	},
	{
		primitive:      "RSA",
		category:       "DigitalSignature",
		importContains: []string{"signerrsa", "rsa"},
		nameContains:   []string{"signerrsa", "rsa"},
	},
	{
		primitive:        "ERC1271",
		category:         "DigitalSignature",
		importContains:   []string{"erc1271", "signaturechecker"},
		inheritsContains: []string{"erc1271"},
		funcContains:     []string{"isvalidsignature"},
	},
	{
		primitive:      "SignatureChecker",
		category:       "DigitalSignature",
		importContains: []string{"signaturechecker"},
	},
	{
		primitive:        "EIP712",
		category:         "TypedDataSigning",
		importContains:   []string{"eip712"},
		inheritsContains: []string{"eip712"},
		funcContains:     []string{"_hashtypeddatav4"},
	},
	{
		primitive:      "MerkleProof",
		category:       "MerkleTree",
		importContains: []string{"merkleproof"},
		nameContains:   []string{"merkle", "proof"},
		funcContains:   []string{"verify", "processproof"},
	},
	{
		primitive:    "Keccak256",
		category:     "HashFunction",
		nameContains: []string{"keccak"},
		funcContains: []string{"keccak256"},
	},
	{
		primitive:    "SHA256",
		category:     "HashFunction",
		nameContains: []string{"sha"},
		funcContains: []string{"sha256"},
	},
}

func (a *CryptoAnalyzer) Analyze(g *graph.Graph) error {
	for _, node := range g.Nodes {
		prims, cats := detectCrypto(node, g)
		if len(prims) > 0 {
			node.Metadata["CryptoPrimitives"] = prims
			node.Metadata["CryptoCategories"] = cats
		}
	}
	return nil
}

// detectCrypto applies all rules to a single node and returns de-duplicated
// primitives and categories.
func detectCrypto(node *graph.Node, g *graph.Graph) ([]string, []string) {
	imports := importList(node)
	inherits := inheritanceList(node, g)
	functions := functionList(node)
	name := strings.ToLower(node.ID)

	primSeen := make(map[string]bool)
	catSeen := make(map[string]bool)
	var prims, cats []string

	for _, rule := range cryptoRules {
		if matched(rule, name, imports, inherits, functions) {
			if !primSeen[rule.primitive] {
				primSeen[rule.primitive] = true
				prims = append(prims, rule.primitive)
			}
			if !catSeen[rule.category] {
				catSeen[rule.category] = true
				cats = append(cats, rule.category)
			}
		}
	}
	return prims, cats
}

func matched(rule cryptoRule, name string, imports, inherits, functions []string) bool {
	for _, sub := range rule.importContains {
		for _, imp := range imports {
			if strings.Contains(strings.ToLower(imp), sub) {
				return true
			}
		}
	}
	for _, sub := range rule.inheritsContains {
		for _, base := range inherits {
			if strings.Contains(strings.ToLower(base), sub) {
				return true
			}
		}
	}
	for _, sub := range rule.nameContains {
		if strings.Contains(name, sub) {
			return true
		}
	}
	for _, sub := range rule.funcContains {
		for _, fn := range functions {
			if strings.Contains(strings.ToLower(fn), sub) {
				return true
			}
		}
	}
	return false
}
