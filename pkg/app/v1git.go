package app

import (
	"github.com/gin-gonic/gin"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
)

func routerGroupV1Git(config *config.Config, group *gin.RouterGroup) *gin.RouterGroup {
	v1Git := group.Group("/git")
	v1Git.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"backend":    "git",
			"apiVersion": "v1",
		})
	})
	//TODO: implement remote git handler
	return v1Git
}
