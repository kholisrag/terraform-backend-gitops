package command

import (
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	k       = koanf.New(".")
	Konfig  = config.Config{}
	cfgPath string
	Logger  *zap.Logger

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
	Logger, _ = zap.NewProduction()
	defer Logger.Sync()

	cobra.OnInitialize(initConfig)
}

func Execute(version string, commit string, buildTime string) (err error) {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "./.terraform-backend-gitops.yaml", "config file path")

	if err = rootCmd.Execute(); err != nil {
		Logger.Sugar().Fatalf("error executing the command: %v", err)
		return err
	}
	return err
}

func initConfig() {
	// Load the configuration
	if err := k.Load(file.Provider(cfgPath), yaml.Parser()); err != nil {
		Logger.Sugar().Warnf("error loading the configuration: %v", err)
	}
	k.Unmarshal("", &Konfig)
	// Pretty Print the configuration
	Logger.Sugar().Debugf("loaded configuration: %+v", Konfig)

	// Customize the log level dynamically
	currentLogLevel := Logger.Level()
	// Logger.Sugar().Infof("current log level: %v", strings.ToLower(currentLogLevel.String()))
	// Logger.Sugar().Infof("config log level: %v", strings.ToLower(Konfig.LogLevel))

	var newLogLevel zapcore.Level
	switch strings.ToLower(Konfig.LogLevel) {
	case "info", "INFO":
		newLogLevel = zapcore.InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		newLogLevel = zapcore.WarnLevel
	case "error", "ERROR", "err", "ERR":
		newLogLevel = zapcore.ErrorLevel
	case "dpanic", "DPANIC":
		newLogLevel = zapcore.DPanicLevel
	case "panic", "PANIC":
		newLogLevel = zapcore.PanicLevel
	case "fatal", "FATAL":
		newLogLevel = zapcore.FatalLevel
	case "debug", "DEBUG":
		newLogLevel = zapcore.DebugLevel
	default:
		newLogLevel = zapcore.InfoLevel
	}

	if newLogLevel.CapitalString() != currentLogLevel.CapitalString() {
		newEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()) // Access encoder config
		newCore := zapcore.NewCore(
			newEncoder,
			zapcore.AddSync(os.Stdout), // Use AddSync
			zap.NewAtomicLevelAt(newLogLevel),
			// ... other options ...
		)

		Logger = Logger.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
			Logger.Sugar().Infof("log level changed to %v", strings.ToLower(newLogLevel.String()))
			return newCore
		}))
	}

	// Check if the root directory is a Git repository
	repo, err := git.PlainOpen(Konfig.Repo.RepoLocal.Path)
	if err != nil {
		Logger.Sugar().Fatalf("the current/configured directory is not a git repository: %v", err)
	} else {
		// go-git check git repository status
		Logger.Info("the current/configured directory is a git repository")
	}

	// Get the repository's root directory
	repoRoot, err := repo.Worktree()
	if err != nil {
		panic(err)
	}
	Logger.Sugar().Infof("repository root: %v", repoRoot.Filesystem.Root())
	gitStatus, _ := repoRoot.Status()
	Logger.Sugar().Debugf("git status: %v", gitStatus.String())
}
