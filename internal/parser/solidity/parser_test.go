package solidity

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/smartbom/smartbom/internal/parser"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "contracts")
	return root
}

func TestParseERC20(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "ERC20Token.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(pf.Contracts) == 0 {
		t.Fatal("expected at least one contract")
	}

	c := findContract(pf, "MyERC20Token")
	if c == nil {
		t.Fatal("contract MyERC20Token not found")
	}
	if c.Kind != "contract" {
		t.Errorf("kind = %q, want contract", c.Kind)
	}
	if len(c.Imports) == 0 {
		t.Error("expected imports")
	}
	if !containsAny(c.Inherits, "ERC20", "Ownable") {
		t.Errorf("inherits = %v; want ERC20 and Ownable", c.Inherits)
	}
	if !containsFn(c.Functions, "mint") {
		t.Error("expected function 'mint'")
	}
	if len(c.Events) == 0 {
		t.Error("expected events")
	}
}

func TestParseOracle(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "ChainlinkOracle.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	c := findContract(pf, "ChainlinkPriceOracle")
	if c == nil {
		t.Fatal("contract ChainlinkPriceOracle not found")
	}
	if !containsImport(c.Imports, "AggregatorV3Interface") {
		t.Errorf("imports = %v; want AggregatorV3Interface", c.Imports)
	}
	if !containsFn(c.Functions, "getLatestPrice") {
		t.Error("expected function getLatestPrice")
	}

	iface := findContract(pf, "IPriceConsumer")
	if iface == nil {
		t.Fatal("interface IPriceConsumer not found")
	}
	if iface.Kind != "interface" {
		t.Errorf("kind = %q, want interface", iface.Kind)
	}
}

func TestParseProxy(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "UUPSProxy.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	c := findContract(pf, "UpgradeableToken")
	if c == nil {
		t.Fatal("contract UpgradeableToken not found")
	}
	if !containsAny(c.Inherits, "UUPSUpgradeable") {
		t.Errorf("inherits = %v; want UUPSUpgradeable", c.Inherits)
	}
}

func TestParseGovernor(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "Governor.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	c := findContract(pf, "MyGovernor")
	if c == nil {
		t.Fatal("contract MyGovernor not found")
	}
	if !containsAny(c.Inherits, "Governor") {
		t.Errorf("inherits = %v; want Governor", c.Inherits)
	}
}

func TestLanguage(t *testing.T) {
	p := New()
	if p.Language() != "solidity" {
		t.Errorf("Language() = %q, want solidity", p.Language())
	}
}

func TestStripComments(t *testing.T) {
	cases := []struct {
		input   string
		inBlock bool
		want    string
	}{
		{"contract A { // comment", false, "contract A { "},
		{"/* block */ contract B {", false, " contract B {"},
		{"inside block comment", true, ""},
		{"end */ of block", true, " of block"},
	}
	for _, c := range cases {
		got, _ := stripComments(c.input, c.inBlock)
		if got != c.want {
			t.Errorf("stripComments(%q, %v) = %q, want %q", c.input, c.inBlock, got, c.want)
		}
	}
}

func TestParseBaseList(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"ERC20, Ownable", []string{"ERC20", "Ownable"}},
		{"ERC20('Token', 'TKN'), Ownable", []string{"ERC20", "Ownable"}},
		{"Governor", []string{"Governor"}},
	}
	for _, c := range cases {
		got := parseBaseList(c.raw)
		if len(got) != len(c.want) {
			t.Errorf("parseBaseList(%q) = %v, want %v", c.raw, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseBaseList(%q)[%d] = %q, want %q", c.raw, i, got[i], c.want[i])
			}
		}
	}
}

// --- helpers -----------------------------------------------------------------

func findContract(pf *parser.ParsedFile, name string) *parser.Contract {
	for i := range pf.Contracts {
		if pf.Contracts[i].Name == name {
			return &pf.Contracts[i]
		}
	}
	return nil
}

func containsAny(slice []string, targets ...string) bool {
	for _, t := range targets {
		for _, s := range slice {
			if s == t {
				return true
			}
		}
	}
	return false
}

func containsFn(fns []parser.Function, name string) bool {
	for _, f := range fns {
		if f.Name == name {
			return true
		}
	}
	return false
}

func containsImport(imports []string, substr string) bool {
	for _, imp := range imports {
		if strings.Contains(imp, substr) {
			return true
		}
	}
	return false
}
