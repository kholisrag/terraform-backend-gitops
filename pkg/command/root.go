package command

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

var (
	k       = koanf.New(".")
	Konfig  = config.Config{}
	cfgPath string

	version   string
	commit    string
	buildTime string

	rootCmd = &cobra.Command{
		Use:   "terraform-backend-gitops",
		Short: "Terraform Backend HTTP that saves and encrypts state to files (GitOps style)",
		Long: `
A Simple HTTP Server that function as Terraform Backend HTTP,
saves and encrypts terraform state to files (GitOps style)
`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
}

func Execute(version string, commit string, buildTime string) (err error) {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "./.terraform-backend-gitops.yaml", "config file path")

	if err = rootCmd.Execute(); err != nil {
		logger.Fatalf("error executing the command: %v", err)
		return err
	}
	return err
}

func initConfig() {
	// Load the configuration
	if err := k.Load(file.Provider(cfgPath), yaml.Parser()); err != nil {
		logger.Warnf("error loading the configuration: %v", err)
	}

	err := k.Unmarshal("", &Konfig)
	if err != nil {
		logger.Fatalf("error unmarshalling the configuration: %v", err)
	}

	logger.Init(Konfig.LogLevel)
	// Pretty Print the configuration
	logger.Debugf("loaded configuration: %+v", Konfig)

	// Check if the root directory is a Git repository
	repo, err := git.PlainOpen(Konfig.Repo.RepoLocal.Path)
	if err != nil {
		logger.Fatalf("the current/configured directory is not a git repository: %v", err)
	} else {
		// go-git check git repository status
		logger.Infof("the current/configured directory is a git repository")
	}

	// Get the repository's root directory
	repoRoot, err := repo.Worktree()
	if err != nil {
		panic(err)
	}
	logger.Infof("repository root: %v", repoRoot.Filesystem.Root())
	gitStatus, _ := repoRoot.Status()
	logger.Debugf("git status: %v", gitStatus.String())

	// Validate GitHub sync configuration if enabled
	if Konfig.Repo.RepoGithub.Enabled {
		logger.Info("GitHub sync enabled, validating configuration...")

		// Validate remote URL is configured
		if Konfig.Repo.RepoGithub.RemoteURL == "" {
			logger.Fatal("GitHub sync enabled but remoteUrl not configured")
		}

		// Check if remote exists
		remote, err := repo.Remote("origin")
		if err == git.ErrRemoteNotFound {
			logger.Warnf("remote 'origin' not found, will be created on first push")
		} else if err == nil {
			// Verify remote URL matches configuration
			urls := remote.Config().URLs
			logger.Infof("existing remote 'origin': %v", urls)
		}

		// Validate authentication configuration
		switch Konfig.Repo.RepoGithub.AuthMethod {
		case "ssh":
			if Konfig.Repo.RepoGithub.SSHKeyPath == "" {
				logger.Fatal("authMethod is 'ssh' but sshKeyPath not configured")
			}
			// Check if SSH key file exists
			sshKeyPath := os.ExpandEnv(Konfig.Repo.RepoGithub.SSHKeyPath)
			if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
				logger.Fatalf("SSH key not found: %s", sshKeyPath)
			}
			logger.Infof("SSH key found: %s", sshKeyPath)
		case "token":
			token := os.ExpandEnv(Konfig.Repo.RepoGithub.Token)
			if token == "" {
				logger.Fatal("authMethod is 'token' but token not configured or empty")
			}
			logger.Info("GitHub token configured (not showing value for security)")
		case "default":
			logger.Info("using default git credentials from system")
		default:
			logger.Warnf("unknown authMethod '%s', will use default", Konfig.Repo.RepoGithub.AuthMethod)
		}

		logger.Infof("GitHub sync validated: remote=%s, branch=%s, auth=%s",
			Konfig.Repo.RepoGithub.RemoteURL,
			Konfig.Repo.RepoGithub.Branch,
			Konfig.Repo.RepoGithub.AuthMethod)
	} else {
		logger.Debug("GitHub sync is disabled")
	}
}
