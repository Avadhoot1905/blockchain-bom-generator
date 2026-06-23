package packages

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file with the exact name in a per-test temp directory.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFromPackageJSON_Dependencies(t *testing.T) {
	src := `{
  "dependencies": {
    "@openzeppelin/contracts": "^4.9.0",
    "hardhat": "^2.19.0"
  },
  "devDependencies": {
    "@nomiclabs/hardhat-ethers": "^2.0.0"
  }
}`
	path := writeFile(t, "package.json", src)
	got, err := fromPackageJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"@openzeppelin/contracts":      "^4.9.0",
		"hardhat":                      "^2.19.0",
		"@nomiclabs/hardhat-ethers":    "^2.0.0",
	}
	for pkg, want := range cases {
		if got[pkg] != want {
			t.Errorf("package %q: got %q, want %q", pkg, got[pkg], want)
		}
	}
}

func TestFromPackageJSON_Empty(t *testing.T) {
	path := writeFile(t, "package.json", `{}`)
	got, err := fromPackageJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestFromFoundryTOML_SimpleVersion(t *testing.T) {
	src := `[profile.default]
src = "src"

[dependencies]
forge-std = "1.6.1"
"@openzeppelin/contracts" = "5.0.0"
`
	path := writeFile(t, "foundry.toml", src)
	got, err := fromFoundryTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["forge-std"] != "1.6.1" {
		t.Errorf("forge-std: got %q, want 1.6.1", got["forge-std"])
	}
	if got["@openzeppelin/contracts"] != "5.0.0" {
		t.Errorf("@openzeppelin/contracts: got %q, want 5.0.0", got["@openzeppelin/contracts"])
	}
}

func TestFromFoundryTOML_TableVersion(t *testing.T) {
	src := `[dependencies]
openzeppelin = { version = "4.9.0", git = "https://github.com/OpenZeppelin/openzeppelin-contracts" }
`
	path := writeFile(t, "foundry.toml", src)
	got, err := fromFoundryTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["openzeppelin"] != "4.9.0" {
		t.Errorf("openzeppelin: got %q, want 4.9.0", got["openzeppelin"])
	}
}

func TestFromFoundryTOML_TagVersion(t *testing.T) {
	src := `[dependencies]
openzeppelin = { git = "https://github.com/OpenZeppelin/openzeppelin-contracts", tag = "v5.0.0" }
`
	path := writeFile(t, "foundry.toml", src)
	got, err := fromFoundryTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["openzeppelin"] != "5.0.0" {
		t.Errorf("tag version: got %q, want 5.0.0 (v-prefix stripped)", got["openzeppelin"])
	}
}

func TestExtractVersions_Merged(t *testing.T) {
	pkgJSON := writeFile(t, "package.json", `{
  "dependencies": { "@openzeppelin/contracts": "^4.9.0" }
}`)
	foundryTOML := writeFile(t, "foundry.toml", `[dependencies]
forge-std = "1.6.1"
`)
	got := ExtractVersions([]string{pkgJSON, foundryTOML})
	if got["@openzeppelin/contracts"] != "^4.9.0" {
		t.Errorf("@openzeppelin/contracts: got %q", got["@openzeppelin/contracts"])
	}
	if got["forge-std"] != "1.6.1" {
		t.Errorf("forge-std: got %q", got["forge-std"])
	}
}

func TestExtractVersions_EmptyList(t *testing.T) {
	got := ExtractVersions(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}
