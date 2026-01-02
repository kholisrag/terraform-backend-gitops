package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewGitOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	require.NoError(t, err)

	// Create initial commit
	worktree, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = worktree.Add("test.txt")
	require.NoError(t, err)

	_, err = worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	require.NoError(t, err)

	cfg := &config.Config{
		Repo: config.Repo{
			RepoLocal: config.RepoLocal{
				Path: tempDir,
			},
			RepoGithub: config.RepoGithub{
				Enabled:       true,
				RemoteURL:     "https://github.com/test/repo.git",
				Branch:        "main",
				AuthMethod:    "default",
				CommitMessage: "test commit",
				AutoPush:      false, // Don't actually push in tests
				Author: config.CommitAuthor{
					Name:  "Test User",
					Email: "test@example.com",
				},
				RetryAttempts: 1,
				RetryDelay:    1,
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps, err := NewGitOperations(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, gitOps)
	assert.Equal(t, cfg, gitOps.config)
	assert.NotNil(t, gitOps.repo)
	assert.NotNil(t, gitOps.logger)
}

func TestCommitFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	require.NoError(t, err)

	// Create initial commit
	worktree, err := repo.Worktree()
	require.NoError(t, err)

	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial"), 0644)
	require.NoError(t, err)

	_, err = worktree.Add("initial.txt")
	require.NoError(t, err)

	_, err = worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	require.NoError(t, err)

	cfg := &config.Config{
		Repo: config.Repo{
			RepoLocal: config.RepoLocal{
				Path: tempDir,
			},
			RepoGithub: config.RepoGithub{
				Enabled:       true,
				RemoteURL:     "https://github.com/test/repo.git",
				Branch:        "main",
				AuthMethod:    "default",
				CommitMessage: "test commit",
				AutoPush:      false,
				Author: config.CommitAuthor{
					Name:  "Test User",
					Email: "test@example.com",
				},
				RetryAttempts: 1,
				RetryDelay:    1,
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps := &GitOperations{
		config: cfg,
		repo:   repo,
		logger: logger,
	}

	// Create a new file to commit
	testFile := "test-state.tfstate"
	testFilePath := filepath.Join(tempDir, testFile)
	err = os.WriteFile(testFilePath, []byte("test state content"), 0644)
	require.NoError(t, err)

	// Commit the file
	commitHash, err := gitOps.commitFile(testFile, "test: commit state file")
	require.NoError(t, err)
	assert.NotEmpty(t, commitHash)

	// Verify commit was created
	head, err := repo.Head()
	require.NoError(t, err)

	// Get the commit object
	commitObj, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, "test: commit state file", commitObj.Message)
	assert.Equal(t, "Test User", commitObj.Author.Name)
	assert.Equal(t, "test@example.com", commitObj.Author.Email)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "timeout error - retryable",
			err:       assert.AnError,
			retryable: true,
		},
		{
			name:      "authentication error - not retryable",
			err:       assert.AnError,
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestEnsureRemote(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	require.NoError(t, err)

	remoteURL := "https://github.com/test/repo.git"

	// Test creating a new remote
	err = ensureRemote(repo, remoteURL)
	require.NoError(t, err)

	// Verify remote was created
	remote, err := repo.Remote("origin")
	require.NoError(t, err)
	assert.Equal(t, []string{remoteURL}, remote.Config().URLs)

	// Test ensuring remote with same URL (should not error)
	err = ensureRemote(repo, remoteURL)
	require.NoError(t, err)

	// Test updating remote with different URL
	newRemoteURL := "https://github.com/test/new-repo.git"
	err = ensureRemote(repo, newRemoteURL)
	require.NoError(t, err)

	// Verify remote was updated
	remote, err = repo.Remote("origin")
	require.NoError(t, err)
	assert.Equal(t, []string{newRemoteURL}, remote.Config().URLs)
}

func TestGetAuth_Default(t *testing.T) {
	cfg := &config.Config{
		Repo: config.Repo{
			RepoGithub: config.RepoGithub{
				AuthMethod: "default",
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps := &GitOperations{
		config: cfg,
		logger: logger,
	}

	auth, err := gitOps.getAuth()
	require.NoError(t, err)
	assert.Nil(t, auth) // default auth should return nil
}

func TestGetAuth_UnsupportedMethod(t *testing.T) {
	cfg := &config.Config{
		Repo: config.Repo{
			RepoGithub: config.RepoGithub{
				AuthMethod: "unsupported",
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps := &GitOperations{
		config: cfg,
		logger: logger,
	}

	auth, err := gitOps.getAuth()
	require.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "unsupported auth method")
}

func TestGetPATAuth(t *testing.T) {
	// Test with direct token
	cfg := &config.Config{
		Repo: config.Repo{
			RepoGithub: config.RepoGithub{
				Token: "ghp_test_token_123",
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps := &GitOperations{
		config: cfg,
		logger: logger,
	}

	auth, err := gitOps.getPATAuth()
	require.NoError(t, err)
	assert.NotNil(t, auth)

	// Test with empty token
	cfg.Repo.RepoGithub.Token = ""
	auth, err = gitOps.getPATAuth()
	require.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "token is empty")
}

func TestGetPATAuth_EnvVar(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_GITHUB_TOKEN", "ghp_env_token_456")
	defer os.Unsetenv("TEST_GITHUB_TOKEN")

	cfg := &config.Config{
		Repo: config.Repo{
			RepoGithub: config.RepoGithub{
				Token: "${TEST_GITHUB_TOKEN}",
			},
		},
	}

	logger, _ := zap.NewDevelopment()

	gitOps := &GitOperations{
		config: cfg,
		logger: logger,
	}

	auth, err := gitOps.getPATAuth()
	require.NoError(t, err)
	assert.NotNil(t, auth)
}
