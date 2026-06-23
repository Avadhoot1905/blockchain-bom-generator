// Package solidity implements a regex-based Solidity source parser.
// It extracts contracts, interfaces, libraries, imports, inheritance chains,
// functions, events, and modifiers without requiring the solc compiler.
// The implementation is conservative: it favours high-confidence extractions
// over 100% grammar coverage. A solc/tree-sitter backend can be swapped in
// by implementing the parser.Parser interface.
package solidity

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"strings"

	"github.com/smartbom/smartbom/internal/parser"
)

// Compile all regular expressions once at package init for performance.
var (
	// File-level metadata: SPDX license identifier and Solidity compiler pragma.
	reSPDX   = regexp.MustCompile(`//\s*SPDX-License-Identifier:\s*(\S+)`)
	rePragma = regexp.MustCompile(`^\s*pragma\s+solidity\s+([^;]+);`)

	// Import patterns (double-quoted, single-quoted, and 'from' variants).
	reImportDouble = regexp.MustCompile(`^\s*import\s+"([^"]+)"`)
	reImportSingle = regexp.MustCompile(`^\s*import\s+'([^']+)'`)
	reImportFrom   = regexp.MustCompile(`from\s+"([^"]+)"`)
	reImportFrom2  = regexp.MustCompile(`from\s+'([^']+)'`)

	// Contract / interface / library / abstract contract declaration.
	// The regex is intentionally lenient about the base-list to support
	// multi-line declarations that are pre-joined by the parser loop.
	reContractDecl = regexp.MustCompile(
		`(?s)(abstract\s+)?(contract|interface|library)\s+(\w+)` +
			`(?:\s+is\s+([^{]+))?\s*\{`,
	)

	// Function declaration.
	reFuncDecl = regexp.MustCompile(
		`^\s*function\s+(\w+)\s*\(` +
			`[^)]*\)\s*` +
			`(public|private|internal|external)?\s*` +
			`(pure|view|payable|nonpayable)?\s*`,
	)

	// Event and modifier declarations.
	reEventDecl    = regexp.MustCompile(`^\s*event\s+(\w+)\s*\(`)
	reModifierDecl = regexp.MustCompile(`^\s*modifier\s+(\w+)\s*[\({]`)
)

// SolidityParser implements parser.Parser for Solidity source files.
type SolidityParser struct{}

// New returns a new SolidityParser.
func New() *SolidityParser { return &SolidityParser{} }

// Language satisfies parser.Parser.
func (p *SolidityParser) Language() string { return "solidity" }

// Parse reads a Solidity file and returns its extracted declarations.
func (p *SolidityParser) Parse(path string) (*parser.ParsedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)
	pf := &parser.ParsedFile{
		Path:       path,
		Language:   "solidity",
		SourceHash: hex.EncodeToString(sum[:]),
	}

	var (
		current       *parser.Contract
		inBlockCmt    bool
		braceDepth    int
		contractDepth int
		stripped      string

		// Multi-line contract declaration accumulator.
		accumulating bool
		accumBuf     strings.Builder
	)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		raw := scanner.Text()

		// SPDX and pragma must be checked on the raw line BEFORE comment stripping,
		// because `// SPDX-License-Identifier: MIT` is entirely a comment and becomes
		// empty after stripping, causing the `continue` below to skip it.
		if pf.License == "" {
			if m := reSPDX.FindStringSubmatch(raw); m != nil {
				pf.License = strings.TrimSpace(m[1])
			}
		}

		stripped, inBlockCmt = stripComments(raw, inBlockCmt)
		trimmed := strings.TrimSpace(stripped)
		if trimmed == "" {
			continue
		}

		if pf.SolidityVersion == "" {
			if m := rePragma.FindStringSubmatch(trimmed); m != nil {
				pf.SolidityVersion = strings.TrimSpace(m[1])
			}
		}

		// Handle multi-line contract declaration accumulation.
		if accumulating {
			accumBuf.WriteString(" ")
			accumBuf.WriteString(trimmed)
			if strings.Contains(trimmed, "{") {
				// We now have the full declaration; parse it.
				m := reContractDecl.FindStringSubmatch(accumBuf.String())
				accumulating = false
				if m != nil {
					current = buildContract(m, path, pf)
					contractDepth = braceDepth
					braceDepth += strings.Count(accumBuf.String(), "{") -
						strings.Count(accumBuf.String(), "}")
					// Don't count the opening brace twice below.
					continue
				}
			}
			// Count braces in accumulator lines too.
			braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			continue
		}

		// Count braces before deciding on the current contract.
		opens := strings.Count(trimmed, "{")
		closes := strings.Count(trimmed, "}")
		oldDepth := braceDepth
		braceDepth += opens - closes

		// File-level import (always at depth 0, before any contract).
		if imp := extractImport(trimmed); imp != "" {
			if current == nil {
				pf.FileImports = append(pf.FileImports, imp)
			} else {
				current.Imports = append(current.Imports, imp)
			}
			continue
		}

		// Contract / interface / library declaration.
		if m := reContractDecl.FindStringSubmatch(trimmed); m != nil {
			if current != nil {
				pf.Contracts = append(pf.Contracts, *current)
			}
			current = buildContract(m, path, pf)
			contractDepth = oldDepth
			continue
		}

		// Start of multi-line declaration: has "contract|interface|library Name is" but no "{".
		if !strings.Contains(trimmed, "{") {
			if isContractDeclStart(trimmed) {
				accumulating = true
				accumBuf.Reset()
				accumBuf.WriteString(trimmed)
				continue
			}
		}

		if current == nil {
			continue
		}

		// Function declaration.
		if m := reFuncDecl.FindStringSubmatch(trimmed); m != nil {
			fn := parser.Function{
				Name:       m[1],
				Visibility: strings.TrimSpace(m[2]),
				Mutability: strings.TrimSpace(m[3]),
			}
			current.Functions = append(current.Functions, fn)
			continue
		}

		// Event declaration.
		if m := reEventDecl.FindStringSubmatch(trimmed); m != nil {
			current.Events = append(current.Events, m[1])
			continue
		}

		// Modifier declaration.
		if m := reModifierDecl.FindStringSubmatch(trimmed); m != nil {
			current.Modifiers = append(current.Modifiers, m[1])
			continue
		}

		// Contract body ended when depth returns to or below entry level.
		if braceDepth <= contractDepth && opens == 0 && closes > 0 {
			pf.Contracts = append(pf.Contracts, *current)
			current = nil
		}
	}

	if current != nil {
		pf.Contracts = append(pf.Contracts, *current)
	}

	return pf, scanner.Err()
}

// buildContract creates a Contract from a reContractDecl match and inherits
// any accumulated file-level imports.
func buildContract(m []string, path string, pf *parser.ParsedFile) *parser.Contract {
	isAbstract := strings.TrimSpace(m[1]) == "abstract"
	kind := m[2]
	name := m[3]
	basesRaw := strings.TrimSpace(m[4])

	if isAbstract {
		kind = "abstract"
	}

	// Seed contract imports with file-level imports, then drain them so
	// subsequent contracts get a fresh copy.
	fileImports := make([]string, len(pf.FileImports))
	copy(fileImports, pf.FileImports)

	c := &parser.Contract{
		Name:       name,
		Kind:       kind,
		SourceFile: path,
		Imports:    fileImports,
	}
	if basesRaw != "" {
		c.Inherits = parseBaseList(basesRaw)
	}
	return c
}

// isContractDeclStart returns true when a line begins a multi-line contract
// declaration (has the contract/interface/library keyword but no opening brace).
var reContractStart = regexp.MustCompile(
	`^\s*(abstract\s+)?(contract|interface|library)\s+\w+`,
)

func isContractDeclStart(line string) bool {
	return reContractStart.MatchString(line)
}

// --- helpers -----------------------------------------------------------------

func extractImport(line string) string {
	if m := reImportDouble.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	if m := reImportSingle.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	if strings.Contains(line, "from") {
		if m := reImportFrom.FindStringSubmatch(line); m != nil {
			return m[1]
		}
		if m := reImportFrom2.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// parseBaseList splits an inheritance list that may contain constructor
// arguments, correctly handling nested parentheses.
// "ERC20('Token', 'SYM'), Ownable" → ["ERC20", "Ownable"]
func parseBaseList(raw string) []string {
	var result []string
	depth := 0
	start := 0

	for i, ch := range raw {
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				if name := extractBaseName(raw[start:i]); name != "" {
					result = append(result, name)
				}
				start = i + 1
			}
		}
	}
	if name := extractBaseName(raw[start:]); name != "" {
		result = append(result, name)
	}
	return result
}

// extractBaseName trims whitespace and strips any constructor arguments.
// "ERC20('name')" → "ERC20", "  Ownable  " → "Ownable"
func extractBaseName(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "("); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	return s
}

// stripComments removes line and block comments from a single line.
// inBlock tracks whether we are currently inside a /* */ block.
func stripComments(line string, inBlock bool) (string, bool) {
	var sb strings.Builder
	i := 0
	for i < len(line) {
		if inBlock {
			idx := strings.Index(line[i:], "*/")
			if idx == -1 {
				return sb.String(), true
			}
			i += idx + 2
			inBlock = false
			continue
		}
		if i+1 < len(line) && line[i] == '/' && line[i+1] == '*' {
			inBlock = true
			i += 2
			continue
		}
		if i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
			break
		}
		sb.WriteByte(line[i])
		i++
	}
	return sb.String(), inBlock
}
