// Package packages extracts dependency names and versions from project manifests
// (package.json for npm/Hardhat projects; foundry.toml for Forge projects).
// Extraction is intentionally simple: no dependency graph resolution, no lockfile
// parsing, no network calls.
package packages

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// ExtractVersions reads all provided manifest paths and returns a merged map of
// package-name → version string.  Later manifests win on conflict.
func ExtractVersions(manifestPaths []string) map[string]string {
	result := make(map[string]string)
	for _, path := range manifestPaths {
		lower := strings.ToLower(path)
		var m map[string]string
		switch {
		case strings.HasSuffix(lower, "package.json"):
			m, _ = fromPackageJSON(path)
		case strings.HasSuffix(lower, "foundry.toml"):
			m, _ = fromFoundryTOML(path)
		}
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// fromPackageJSON parses the "dependencies" and "devDependencies" sections of a
// package.json file and returns a name → version map.
func fromPackageJSON(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range raw.Dependencies {
		result[k] = v
	}
	for k, v := range raw.DevDependencies {
		result[k] = v
	}
	return result, nil
}

// reFoundryDep matches lines in [dependencies] sections of foundry.toml in the
// forms produced by both the legacy `[dependencies]` block and soldeer:
//
//	"@openzeppelin/contracts" = "5.0.0"
//	forge-std = "1.6.1"
//	openzeppelin = { version = "4.9.0", git = "..." }
//	openzeppelin = { git = "...", tag = "v4.9.0" }
var reFoundrySimple = regexp.MustCompile(`^\s*"?([^"=\s]+)"?\s*=\s*"([^"]+)"`)
var reFoundryTagged = regexp.MustCompile(`tag\s*=\s*"([^"]+)"`)
var reFoundryVersion = regexp.MustCompile(`version\s*=\s*"([^"]+)"`)

// fromFoundryTOML parses the [dependencies] section of a foundry.toml file
// using line-by-line regex matching (avoids adding a TOML library dependency).
func fromFoundryTOML(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	inDeps := false

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)

		// Section headers.
		if strings.HasPrefix(trimmed, "[") {
			inDeps = strings.TrimSpace(trimmed) == "[dependencies]"
			continue
		}
		if !inDeps || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Simple form: name = "version"
		if m := reFoundrySimple.FindStringSubmatch(trimmed); m != nil {
			result[m[1]] = m[2]
			continue
		}

		// Table form with version or tag key: name = { version = "x", ... }
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(trimmed[:eqIdx]), `"`)
		rest := trimmed[eqIdx+1:]

		if mv := reFoundryVersion.FindStringSubmatch(rest); mv != nil {
			result[name] = mv[1]
		} else if mt := reFoundryTagged.FindStringSubmatch(rest); mt != nil {
			result[name] = strings.TrimPrefix(mt[1], "v")
		}
	}
	return result, nil
}
