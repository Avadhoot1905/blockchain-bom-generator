package git

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository holds the location of a cloned repository on disk.
type Repository struct {
	URL    string
	Path   string
	Branch string
}

// RepositoryManager defines how repositories are acquired and cleaned up.
type RepositoryManager interface {
	Clone(url string) (*Repository, error)
	CloneWithRef(url, ref string) (*Repository, error)
	Cleanup(repo *Repository) error
}

// Manager is the concrete implementation backed by go-git.
type Manager struct {
	workDir string // parent directory for all clones; temp dir if empty
}

// NewManager creates a Manager that clones into workDir.
// Pass an empty string to use the system temp directory.
func NewManager(workDir string) *Manager {
	return &Manager{workDir: workDir}
}

// Clone clones the default branch of url into a unique subdirectory.
func (m *Manager) Clone(url string) (*Repository, error) {
	return m.CloneWithRef(url, "")
}

// CloneWithRef clones a specific branch or tag. Pass an empty ref for HEAD.
func (m *Manager) CloneWithRef(url, ref string) (*Repository, error) {
	destDir, err := m.resolveDestDir(url)
	if err != nil {
		return nil, err
	}

	slog.Info("cloning repository", "url", url, "dest", destDir)

	opts := &gogit.CloneOptions{
		URL:      url,
		Progress: os.Stderr,
		Depth:    1,
	}
	if ref != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(ref)
		opts.SingleBranch = true
	}

	_, err = gogit.PlainClone(destDir, false, opts)
	if err != nil {
		// If the directory already exists with a valid repo, reuse it.
		if isExistingRepo(destDir) {
			slog.Info("repository already cloned, reusing", "path", destDir)
			return &Repository{URL: url, Path: destDir, Branch: ref}, nil
		}
		return nil, fmt.Errorf("clone %s: %w", url, err)
	}

	slog.Info("clone complete", "path", destDir)
	return &Repository{URL: url, Path: destDir, Branch: ref}, nil
}

// Cleanup removes the cloned repository from disk.
func (m *Manager) Cleanup(repo *Repository) error {
	if repo == nil || repo.Path == "" {
		return nil
	}
	slog.Info("cleaning up repository", "path", repo.Path)
	return os.RemoveAll(repo.Path)
}

func (m *Manager) resolveDestDir(url string) (string, error) {
	base := m.workDir
	if base == "" {
		var err error
		base, err = os.MkdirTemp("", "smartbom-*")
		if err != nil {
			return "", fmt.Errorf("create temp dir: %w", err)
		}
	} else {
		if err := os.MkdirAll(base, 0o755); err != nil {
			return "", fmt.Errorf("create work dir: %w", err)
		}
	}
	return filepath.Join(base, repoName(url)), nil
}

// repoName derives a filesystem-safe directory name from a repository URL.
func repoName(url string) string {
	name := filepath.Base(url)
	name = strings.TrimSuffix(name, ".git")
	// Replace any characters that could be problematic on Windows/macOS.
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '-'
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "repo"
	}
	return name
}

func isExistingRepo(path string) bool {
	_, err := gogit.PlainOpen(path)
	return err == nil
}
