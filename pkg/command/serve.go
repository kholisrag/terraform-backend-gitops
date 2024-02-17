package command

import (
	"github.com/kholisrag/terraform-backend-gitops/pkg/app"
	"github.com/spf13/cobra"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP Server",
		Run: func(cmd *cobra.Command, args []string) {
			// Start the Apps
			Konfig.Build.Version = version
			Konfig.Build.CommitHash = commit
			Konfig.Build.BuildTime = buildTime

			s := app.NewApp(Logger, &Konfig)
			s.Run(Konfig.Server.Address)
		},
	}
)

func init() {
	rootCmd.AddCommand(serveCmd)
}
