package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileScanner(t *testing.T) {
	root := t.TempDir()

	// Create a realistic directory layout.
	mustMkdir := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(p, content string) {
		t.Helper()
		full := filepath.Join(root, p)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustMkdir("contracts")
	mustMkdir("node_modules/openzeppelin") // should be skipped
	mustMkdir("scripts")

	mustWrite("contracts/Token.sol", "// solidity")
	mustWrite("contracts/Vault.sol", "// solidity")
	mustWrite("contracts/Oracle.vy", "# vyper")
	mustWrite("foundry.toml", "[profile.default]")
	mustWrite("package.json", "{}")
	mustWrite("hardhat.config.ts", "")
	mustWrite("node_modules/openzeppelin/ERC20.sol", "// should not appear")

	s := NewFileScanner()
	project, err := s.Scan(root)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(project.SolidityFiles) != 2 {
		t.Errorf("expected 2 solidity files, got %d", len(project.SolidityFiles))
	}
	if len(project.VyperFiles) != 1 {
		t.Errorf("expected 1 vyper file, got %d", len(project.VyperFiles))
	}
	if len(project.ConfigFiles) != 3 {
		t.Errorf("expected 3 config files, got %d", len(project.ConfigFiles))
	}
}

func TestShouldSkipDir(t *testing.T) {
	cases := []struct {
		name string
		skip bool
	}{
		{"node_modules", true},
		{".git", true},
		{"artifacts", true},
		{"contracts", false},
		{"src", false},
	}
	for _, c := range cases {
		if shouldSkipDir(c.name) != c.skip {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", c.name, !c.skip, c.skip)
		}
	}
}
