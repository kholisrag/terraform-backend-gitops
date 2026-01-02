package storage

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"go.uber.org/zap"

	appconfig "github.com/kholisrag/terraform-backend-gitops/pkg/config"
)

// GitOperations handles git commit and push operations for state files
type GitOperations struct {
	config *appconfig.Config
	repo   *git.Repository
	logger *zap.Logger
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(cfg *appconfig.Config, logger *zap.Logger) (*GitOperations, error) {
	// Open the repository
	repo, err := git.PlainOpen(cfg.Repo.RepoLocal.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Ensure remote is configured
	if err := ensureRemote(repo, cfg.Repo.RepoGithub.RemoteURL); err != nil {
		return nil, fmt.Errorf("failed to ensure remote: %w", err)
	}

	return &GitOperations{
		config: cfg,
		repo:   repo,
		logger: logger,
	}, nil
}

// CommitAndPush commits a file and pushes to the remote repository
func (g *GitOperations) CommitAndPush(filePath, commitMessage string) error {
	// Commit the file
	commitHash, err := g.commitFile(filePath, commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit file: %w", err)
	}

	g.logger.Info("committed file to git",
		zap.String("file", filePath),
		zap.String("commit", commitHash))

	// Push to remote with retry
	if g.config.Repo.RepoGithub.AutoPush {
		if err := g.retryOperation(g.pushToRemote); err != nil {
			return fmt.Errorf("failed to push to remote: %w", err)
		}

		g.logger.Info("pushed to remote successfully",
			zap.String("remote", g.config.Repo.RepoGithub.RemoteURL),
			zap.String("branch", g.config.Repo.RepoGithub.Branch))
	}

	return nil
}

// commitFile stages and commits a specific file
func (g *GitOperations) commitFile(filePath, commitMessage string) (string, error) {
	worktree, err := g.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Stage the specific file
	_, err = worktree.Add(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stage file %s: %w", filePath, err)
	}

	// Create commit
	author := &object.Signature{
		Name:  g.config.Repo.RepoGithub.Author.Name,
		Email: g.config.Repo.RepoGithub.Author.Email,
		When:  time.Now(),
	}

	commit, err := worktree.Commit(commitMessage, &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	return commit.String(), nil
}

// pushToRemote pushes to the configured remote repository
func (g *GitOperations) pushToRemote() error {
	auth, err := g.getAuth()
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Create refspec for the configured branch
	refSpec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s",
		g.config.Repo.RepoGithub.Branch,
		g.config.Repo.RepoGithub.Branch))

	err = g.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpec},
		Auth:       auth,
	})

	if err != nil {
		// git.NoErrAlreadyUpToDate is not an error
		if err == git.NoErrAlreadyUpToDate {
			g.logger.Debug("repository already up to date")
			return nil
		}
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

// getAuth returns the appropriate authentication method based on configuration
func (g *GitOperations) getAuth() (transport.AuthMethod, error) {
	switch g.config.Repo.RepoGithub.AuthMethod {
	case "ssh":
		return g.getSSHAuth()
	case "token":
		return g.getPATAuth()
	case "default":
		// Use system git credentials (nil auth)
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", g.config.Repo.RepoGithub.AuthMethod)
	}
}

// getSSHAuth creates SSH authentication
func (g *GitOperations) getSSHAuth() (transport.AuthMethod, error) {
	sshKeyPath := g.config.Repo.RepoGithub.SSHKeyPath

	// Expand environment variables and home directory
	sshKeyPath = os.ExpandEnv(sshKeyPath)
	if strings.HasPrefix(sshKeyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		sshKeyPath = strings.Replace(sshKeyPath, "~", homeDir, 1)
	}

	// Create SSH public keys auth
	publicKeys, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key from %s: %w", sshKeyPath, err)
	}

	return publicKeys, nil
}

// getPATAuth creates Personal Access Token authentication
func (g *GitOperations) getPATAuth() (transport.AuthMethod, error) {
	token := g.config.Repo.RepoGithub.Token

	// Expand environment variables
	token = os.ExpandEnv(token)

	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// GitHub expects username "git" for PAT auth
	return &http.BasicAuth{
		Username: "git",
		Password: token,
	}, nil
}

// retryOperation retries an operation with exponential backoff
func (g *GitOperations) retryOperation(operation func() error) error {
	maxAttempts := g.config.Repo.RepoGithub.RetryAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	baseDelay := time.Duration(g.config.Repo.RepoGithub.RetryDelay) * time.Second
	if baseDelay <= 0 {
		baseDelay = 5 * time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable (network errors, temporary failures)
		if !isRetryableError(err) {
			g.logger.Error("non-retryable error, aborting",
				zap.Error(err),
				zap.Int("attempt", attempt))
			return err
		}

		// Don't retry on last attempt
		if attempt < maxAttempts {
			// Exponential backoff: delay * (2 ^ (attempt - 1))
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			g.logger.Warn("operation failed, retrying",
				zap.Error(err),
				zap.Int("attempt", attempt),
				zap.Int("maxAttempts", maxAttempts),
				zap.Duration("retryAfter", delay))
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, lastErr)
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Non-retryable errors
	nonRetryable := []string{
		"authentication required",
		"authentication failed",
		"permission denied",
		"non-fast-forward",
		"conflict",
	}

	for _, msg := range nonRetryable {
		if strings.Contains(strings.ToLower(errStr), msg) {
			return false
		}
	}

	// Retryable errors (network issues, temporary failures)
	retryable := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"no route to host",
		"temporary failure",
		"network is unreachable",
	}

	for _, msg := range retryable {
		if strings.Contains(strings.ToLower(errStr), msg) {
			return true
		}
	}

	// Default: retry on unknown errors
	return true
}

// ensureRemote ensures the remote "origin" exists and matches the configured URL
func ensureRemote(repo *git.Repository, remoteURL string) error {
	remote, err := repo.Remote("origin")
	if err == git.ErrRemoteNotFound {
		// Create the remote
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteURL},
		})
		if err != nil {
			return fmt.Errorf("failed to create remote: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	// Verify remote URL matches configuration
	urls := remote.Config().URLs
	if len(urls) == 0 || urls[0] != remoteURL {
		// Update remote URL
		err = repo.DeleteRemote("origin")
		if err != nil {
			return fmt.Errorf("failed to delete old remote: %w", err)
		}

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteURL},
		})
		if err != nil {
			return fmt.Errorf("failed to recreate remote: %w", err)
		}
	}

	return nil
}
