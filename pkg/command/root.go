package command

import (
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
}
