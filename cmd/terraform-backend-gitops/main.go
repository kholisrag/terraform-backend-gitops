package main

import (
	"github.com/kholisrag/terraform-backend-gitops/pkg/command"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
	"go.uber.org/zap"
)

var (
	version string
	commit  string
	build   string
)

func main() {
	err := command.Execute(version, commit, build)
	if err != nil {
		logger.Fatal("failed to start", zap.Error(err))
	}
}
