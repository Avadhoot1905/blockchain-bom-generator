package solidity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSPDXLicenseExtraction verifies MIT, Apache-2.0, and GPL-3.0 detection.
func TestSPDXLicenseExtraction(t *testing.T) {
	cases := []struct {
		license string
		src     string
	}{
		{"MIT", "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\ncontract A {}"},
		{"Apache-2.0", "// SPDX-License-Identifier: Apache-2.0\npragma solidity ^0.8.0;\ncontract B {}"},
		{"GPL-3.0", "// SPDX-License-Identifier: GPL-3.0\npragma solidity ^0.8.0;\ncontract C {}"},
		{"BSD-3-Clause", "// SPDX-License-Identifier: BSD-3-Clause\npragma solidity ^0.8.0;\ncontract D {}"},
	}

	p := New()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.license, func(t *testing.T) {
			path := writeTemp(t, tc.src)
			pf, err := p.Parse(path)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if pf.License != tc.license {
				t.Errorf("License = %q, want %q", pf.License, tc.license)
			}
		})
	}
}

// TestSPDXFromExistingFixture checks extraction on the real ERC20Token.sol fixture.
func TestSPDXFromExistingFixture(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "ERC20Token.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if pf.License != "MIT" {
		t.Errorf("License = %q, want MIT", pf.License)
	}
}

// TestSolidityVersionExtraction verifies single, range, and caret pragma forms.
func TestSolidityVersionExtraction(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantSub string // substring expected in the extracted version
	}{
		{"caret", "pragma solidity ^0.8.20;\ncontract A {}", "0.8.20"},
		{"range", "pragma solidity >=0.8.0 <0.9.0;\ncontract A {}", "0.8.0"},
		{"exact", "pragma solidity 0.8.19;\ncontract A {}", "0.8.19"},
	}

	p := New()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.src)
			pf, err := p.Parse(path)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !strings.Contains(pf.SolidityVersion, tc.wantSub) {
				t.Errorf("SolidityVersion = %q, want to contain %q", pf.SolidityVersion, tc.wantSub)
			}
		})
	}
}

// TestSolidityVersionFromFixture checks extraction on the real UUPSProxy.sol fixture.
func TestSolidityVersionFromFixture(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "UUPSProxy.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !strings.Contains(pf.SolidityVersion, "0.8.20") {
		t.Errorf("SolidityVersion = %q, want to contain 0.8.20", pf.SolidityVersion)
	}
}

// TestSourceHashDeterminism verifies that SHA-256 output is stable across calls.
func TestSourceHashDeterminism(t *testing.T) {
	p := New()
	path := filepath.Join(testdataDir(), "ERC20Token.sol")

	pf1, err := p.Parse(path)
	if err != nil {
		t.Fatalf("first parse error: %v", err)
	}
	pf2, err := p.Parse(path)
	if err != nil {
		t.Fatalf("second parse error: %v", err)
	}

	if pf1.SourceHash == "" {
		t.Fatal("SourceHash should not be empty")
	}
	if pf1.SourceHash != pf2.SourceHash {
		t.Errorf("hashes differ: %q vs %q", pf1.SourceHash, pf2.SourceHash)
	}
}

// TestSourceHashLength verifies SHA-256 produces a 64-char hex string.
func TestSourceHashLength(t *testing.T) {
	p := New()
	pf, err := p.Parse(filepath.Join(testdataDir(), "ERC20Token.sol"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(pf.SourceHash) != 64 {
		t.Errorf("SourceHash length = %d, want 64 (SHA-256 hex)", len(pf.SourceHash))
	}
}

// TestDistinctFilesHaveDifferentHashes verifies two different files produce
// different digests.
func TestDistinctFilesHaveDifferentHashes(t *testing.T) {
	p := New()
	pf1, _ := p.Parse(filepath.Join(testdataDir(), "ERC20Token.sol"))
	pf2, _ := p.Parse(filepath.Join(testdataDir(), "UUPSProxy.sol"))
	if pf1.SourceHash == pf2.SourceHash {
		t.Error("distinct files should produce different hashes")
	}
}

// writeTemp writes src to a temporary .sol file and returns its path.
func writeTemp(t *testing.T, src string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.sol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(src); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}
