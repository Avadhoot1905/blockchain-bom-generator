package parser

// ParsedFile is the output of a single source file parse.
// It is intentionally language-agnostic; parsers for Vyper, Rust, or Move
// produce the same type, allowing the graph builder to be language-unaware.
type ParsedFile struct {
	Path            string
	Language        string
	License         string // SPDX-License-Identifier value (e.g. "MIT", "Apache-2.0")
	SolidityVersion string // pragma solidity constraint (e.g. "^0.8.20", ">=0.8.0 <0.9.0")
	SourceHash      string // hex-encoded SHA-256 of the raw file bytes
	FileImports     []string // imports declared at file scope (before any contract)
	Contracts       []Contract
}

// Contract represents a single top-level declaration extracted from a source file.
type Contract struct {
	Name       string
	Kind       string // "contract", "interface", "library", "abstract"
	Imports    []string
	Inherits   []string
	Functions  []Function
	Events     []string
	Modifiers  []string
	SourceFile string
}

// Function holds a parsed function signature.
type Function struct {
	Name       string
	Visibility string
	Mutability string
}

// Parser is the interface implemented by all language-specific parsers.
// Implementations must be stateless and safe for concurrent use.
type Parser interface {
	// Parse reads a source file and returns its extracted declarations.
	Parse(path string) (*ParsedFile, error)

	// Language returns the language this parser handles (e.g. "solidity").
	Language() string
}
