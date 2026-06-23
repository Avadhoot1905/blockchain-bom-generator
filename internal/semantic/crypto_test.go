package semantic

import (
	"testing"

	"github.com/smartbom/smartbom/internal/graph"
)

// hasPrimitive checks that node.Metadata["CryptoPrimitives"] contains want.
func hasPrimitive(node *graph.Node, want string) bool {
	prims, ok := node.Metadata["CryptoPrimitives"].([]string)
	if !ok {
		return false
	}
	for _, p := range prims {
		if p == want {
			return true
		}
	}
	return false
}

// hasCategory checks that node.Metadata["CryptoCategories"] contains want.
func hasCategory(node *graph.Node, want string) bool {
	cats, ok := node.Metadata["CryptoCategories"].([]string)
	if !ok {
		return false
	}
	for _, c := range cats {
		if c == want {
			return true
		}
	}
	return false
}

// ─── Positive tests ────────────────────────────────────────────────────────

func TestCryptoAnalyzer_ECDSA_ByImport(t *testing.T) {
	node := makeNode("MySigner", "contract", nil)
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/ECDSA.sol",
	}
	g := buildGraph(node)
	if err := (&CryptoAnalyzer{}).Analyze(g); err != nil {
		t.Fatal(err)
	}
	if !hasPrimitive(node, "ECDSA") {
		t.Error("expected ECDSA primitive")
	}
	if !hasCategory(node, "DigitalSignature") {
		t.Error("expected DigitalSignature category")
	}
}

func TestCryptoAnalyzer_ECDSA_ByName(t *testing.T) {
	node := makeNode("SignatureValidator", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "ECDSA") {
		t.Error("expected ECDSA via name containing 'signature'")
	}
}

func TestCryptoAnalyzer_RSA_ByImport(t *testing.T) {
	node := makeNode("Bridge", "contract", nil)
	node.Metadata["Imports"] = []string{"openzeppelin/utils/cryptography/SignerRSA.sol"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "RSA") {
		t.Error("expected RSA primitive from SignerRSA.sol import")
	}
}

func TestCryptoAnalyzer_RSA_ByName(t *testing.T) {
	node := makeNode("SignerRSA", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "RSA") {
		t.Error("expected RSA primitive from contract name SignerRSA")
	}
	if !hasCategory(node, "DigitalSignature") {
		t.Error("expected DigitalSignature category")
	}
}

func TestCryptoAnalyzer_ERC1271_ByInheritance(t *testing.T) {
	node := makeNode("WalletVerifier", "contract", []string{"ERC1271"})
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "ERC1271") {
		t.Error("expected ERC1271 primitive from inheritance")
	}
	if !hasCategory(node, "DigitalSignature") {
		t.Error("expected DigitalSignature category")
	}
}

func TestCryptoAnalyzer_ERC1271_ByImport(t *testing.T) {
	node := makeNode("Wallet", "contract", nil)
	node.Metadata["Imports"] = []string{"@openzeppelin/contracts/interfaces/IERC1271.sol"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "ERC1271") {
		t.Error("expected ERC1271 primitive from import")
	}
}

func TestCryptoAnalyzer_ERC1271_ByFunction(t *testing.T) {
	node := makeNode("SmartWallet", "contract", nil)
	node.Metadata["Functions"] = []string{"isValidSignature", "execute"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "ERC1271") {
		t.Error("expected ERC1271 primitive from function isValidSignature")
	}
}

func TestCryptoAnalyzer_EIP712_ByImport(t *testing.T) {
	node := makeNode("PermitToken", "contract", nil)
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/EIP712.sol",
	}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "EIP712") {
		t.Error("expected EIP712 primitive from import")
	}
	if !hasCategory(node, "TypedDataSigning") {
		t.Error("expected TypedDataSigning category")
	}
}

func TestCryptoAnalyzer_EIP712_ByInheritance(t *testing.T) {
	node := makeNode("TypedData", "contract", []string{"EIP712", "Ownable"})
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "EIP712") {
		t.Error("expected EIP712 primitive from inheritance")
	}
}

func TestCryptoAnalyzer_EIP712_ByFunction(t *testing.T) {
	node := makeNode("OrderBook", "contract", nil)
	node.Metadata["Functions"] = []string{"_hashTypedDataV4", "fillOrder"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "EIP712") {
		t.Error("expected EIP712 primitive from function _hashTypedDataV4")
	}
}

func TestCryptoAnalyzer_MerkleProof_ByImport(t *testing.T) {
	node := makeNode("Airdrop", "contract", nil)
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/MerkleProof.sol",
	}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "MerkleProof") {
		t.Error("expected MerkleProof primitive from import")
	}
	if !hasCategory(node, "MerkleTree") {
		t.Error("expected MerkleTree category")
	}
}

func TestCryptoAnalyzer_MerkleProof_ByName(t *testing.T) {
	node := makeNode("MerkleDistributor", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "MerkleProof") {
		t.Error("expected MerkleProof primitive from name containing 'merkle'")
	}
}

func TestCryptoAnalyzer_MerkleProof_ByFunction(t *testing.T) {
	node := makeNode("TokenClaim", "contract", nil)
	node.Metadata["Functions"] = []string{"verify", "claim"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "MerkleProof") {
		t.Error("expected MerkleProof primitive from function 'verify'")
	}
}

func TestCryptoAnalyzer_MerkleProof_ByProcessProof(t *testing.T) {
	node := makeNode("TokenClaim", "contract", nil)
	node.Metadata["Functions"] = []string{"processProof"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "MerkleProof") {
		t.Error("expected MerkleProof primitive from function 'processProof'")
	}
}

func TestCryptoAnalyzer_Keccak256_ByName(t *testing.T) {
	node := makeNode("KeccakHasher", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "Keccak256") {
		t.Error("expected Keccak256 primitive from name containing 'keccak'")
	}
	if !hasCategory(node, "HashFunction") {
		t.Error("expected HashFunction category")
	}
}

func TestCryptoAnalyzer_Keccak256_ByFunction(t *testing.T) {
	node := makeNode("Hasher", "contract", nil)
	node.Metadata["Functions"] = []string{"keccak256"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "Keccak256") {
		t.Error("expected Keccak256 primitive from function named keccak256")
	}
}

func TestCryptoAnalyzer_SHA256_ByName(t *testing.T) {
	node := makeNode("SHA256Wrapper", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "SHA256") {
		t.Error("expected SHA256 primitive from name containing 'SHA'")
	}
	if !hasCategory(node, "HashFunction") {
		t.Error("expected HashFunction category")
	}
}

func TestCryptoAnalyzer_SHA256_ByFunction(t *testing.T) {
	node := makeNode("Hasher", "contract", nil)
	node.Metadata["Functions"] = []string{"sha256"}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if !hasPrimitive(node, "SHA256") {
		t.Error("expected SHA256 primitive from function named sha256")
	}
}

// ─── Negative tests ────────────────────────────────────────────────────────

func TestCryptoAnalyzer_NoSignal(t *testing.T) {
	node := makeNode("SimpleStorage", "contract", nil)
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)

	if _, ok := node.Metadata["CryptoPrimitives"]; ok {
		t.Error("SimpleStorage should not have CryptoPrimitives")
	}
	if _, ok := node.Metadata["CryptoCategories"]; ok {
		t.Error("SimpleStorage should not have CryptoCategories")
	}
}

func TestCryptoAnalyzer_GenericERC20_NotTagged(t *testing.T) {
	node := makeNode("MyToken", "contract", []string{"ERC20", "Ownable"})
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)
	if hasPrimitive(node, "ECDSA") {
		t.Error("plain ERC20 should not be tagged with ECDSA")
	}
}

// ─── Multi-primitive test ──────────────────────────────────────────────────

func TestCryptoAnalyzer_MultiPrimitive(t *testing.T) {
	node := makeNode("PermitVault", "contract", []string{"EIP712"})
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/ECDSA.sol",
		"@openzeppelin/contracts/utils/cryptography/MerkleProof.sol",
	}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)

	for _, want := range []string{"ECDSA", "EIP712", "MerkleProof"} {
		if !hasPrimitive(node, want) {
			t.Errorf("expected primitive %q", want)
		}
	}
}

// ─── Pipeline integration ──────────────────────────────────────────────────

func TestDefaultPipeline_IncludesCryptoAnalyzer(t *testing.T) {
	node := makeNode("ECDSAVerifier", "contract", []string{"EIP712"})
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/ECDSA.sol",
	}
	g := buildGraph(node)

	if err := DefaultPipeline().Run(g); err != nil {
		t.Fatal(err)
	}
	if !hasPrimitive(node, "ECDSA") {
		t.Error("DefaultPipeline should run CryptoAnalyzer (ECDSA)")
	}
	if !hasPrimitive(node, "EIP712") {
		t.Error("DefaultPipeline should run CryptoAnalyzer (EIP712)")
	}
}

// TestCryptoAnalyzer_MetadataCreation verifies the metadata keys exist.
func TestCryptoAnalyzer_MetadataCreation(t *testing.T) {
	node := makeNode("MerkleAirdrop", "contract", nil)
	node.Metadata["Imports"] = []string{
		"@openzeppelin/contracts/utils/cryptography/MerkleProof.sol",
	}
	g := buildGraph(node)
	_ = (&CryptoAnalyzer{}).Analyze(g)

	if _, ok := node.Metadata["CryptoPrimitives"]; !ok {
		t.Error("CryptoPrimitives key should be set")
	}
	if _, ok := node.Metadata["CryptoCategories"]; !ok {
		t.Error("CryptoCategories key should be set")
	}
	prims, _ := node.Metadata["CryptoPrimitives"].([]string)
	if len(prims) == 0 {
		t.Error("CryptoPrimitives should not be empty")
	}
	cats, _ := node.Metadata["CryptoCategories"].([]string)
	if len(cats) == 0 {
		t.Error("CryptoCategories should not be empty")
	}
}
