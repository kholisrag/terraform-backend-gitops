package main

import (
	"github.com/kholisrag/terraform-backend-gitops/pkg/command"
)

var (
	version string
	commit  string
	build   string
)

func main() {
	command.Execute(version, commit, build)
}
