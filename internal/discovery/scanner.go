package discovery

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
)

// Project describes the blockchain-related files found inside a repository.
type Project struct {
	RootPath      string
	SolidityFiles []string
	VyperFiles    []string
	RustFiles     []string // Cargo.toml files
	MoveFiles     []string // Move.toml files
	ConfigFiles   []string // foundry.toml, hardhat.config.*, package.json
}

// Scanner discovers blockchain source files inside a repository.
type Scanner interface {
	Scan(rootPath string) (*Project, error)
}

// FileScanner walks the filesystem to discover relevant files.
type FileScanner struct {
	// MaxDepth limits how deep the walk descends; 0 means unlimited.
	MaxDepth int
}

// NewFileScanner creates a FileScanner with unlimited depth.
func NewFileScanner() *FileScanner {
	return &FileScanner{}
}

// Scan walks rootPath and classifies files by extension / name.
func (s *FileScanner) Scan(rootPath string) (*Project, error) {
	project := &Project{RootPath: rootPath}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			if s.MaxDepth > 0 {
				rel, _ := filepath.Rel(rootPath, path)
				depth := strings.Count(rel, string(filepath.Separator))
				if depth >= s.MaxDepth {
					return filepath.SkipDir
				}
			}
			return nil
		}

		name := d.Name()
		lower := strings.ToLower(name)

		switch {
		case strings.HasSuffix(lower, ".sol"):
			project.SolidityFiles = append(project.SolidityFiles, path)
		case strings.HasSuffix(lower, ".vy"):
			project.VyperFiles = append(project.VyperFiles, path)
		case lower == "cargo.toml":
			project.RustFiles = append(project.RustFiles, path)
		case lower == "move.toml":
			project.MoveFiles = append(project.MoveFiles, path)
		case isConfigFile(lower):
			project.ConfigFiles = append(project.ConfigFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("discovery complete",
		"solidity", len(project.SolidityFiles),
		"vyper", len(project.VyperFiles),
		"rust", len(project.RustFiles),
		"move", len(project.MoveFiles),
		"config", len(project.ConfigFiles),
	)
	return project, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".cache", "artifacts", "cache",
		"out", "lib", "dist", "build", ".foundry":
		return true
	}
	return false
}

func isConfigFile(name string) bool {
	if name == "package.json" || name == "foundry.toml" {
		return true
	}
	if strings.HasPrefix(name, "hardhat.config.") {
		return true
	}
	return false
}

// Summary returns a human-readable count of discovered files.
func (p *Project) Summary() string {
	var sb strings.Builder
	sb.WriteString("Discovery summary:\n")
	sb.WriteString("  Solidity: " + intStr(len(p.SolidityFiles)) + "\n")
	sb.WriteString("  Vyper:    " + intStr(len(p.VyperFiles)) + "\n")
	sb.WriteString("  Rust:     " + intStr(len(p.RustFiles)) + "\n")
	sb.WriteString("  Move:     " + intStr(len(p.MoveFiles)) + "\n")
	sb.WriteString("  Config:   " + intStr(len(p.ConfigFiles)) + "\n")
	return sb.String()
}

func intStr(n int) string {
	return fmt.Sprintf("%d", n)
}
