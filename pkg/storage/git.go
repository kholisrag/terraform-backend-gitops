package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"go.uber.org/zap"
)

const Name = "git"

type Git struct {
	Path string `koanf:"path"`
	Root string `koanf:"root,omitempty"`
}

func NewGitStorage(path string, logger *zap.Logger) *Git {
	root, err := getGitRepoRoot(path)
	if err != nil {
		logger.Sugar().Errorf("failed to get git repository root: %v", err)
	}

	return &Git{
		Path: path,
		Root: root,
	}
}

func (g *Git) CheckGitRepo() (valid bool, path string, err error) {
	var isValid bool

	// Check if the repository path exists
	if _, err := os.Stat(g.Path); os.IsNotExist(err) {
		return false, g.Path, errors.New("git repository does not exist")
	}

	// Open the repository
	_, err = git.PlainOpen(g.Path)
	if err != nil {
		return false, g.Path, fmt.Errorf("failed to open git repository: %v", err)
	}

	if err == nil {
		isValid = true
	}

	return isValid, g.Root, nil
}

func getGitRepoRoot(path string) (root string, err error) {
	// Open the repository
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to open git repository worktree: %v", err)
	}

	return worktree.Filesystem.Root(), nil
}

func (g *Git) WriteStateFile(file string, content []byte) error {
	// Create the directory if it does not exist using filepath
	dir := filepath.Dir(fmt.Sprintf("%s/%s", g.Root, file))
	err := os.MkdirAll(dir, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("failed to create directory %v: %v", dir, err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", g.Root, file), content, 0640)
	if err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	return nil
}
