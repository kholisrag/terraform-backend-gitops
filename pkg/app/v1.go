package app

import (
	"github.com/gin-gonic/gin"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
)

func routerGroupV1(config *config.Config, group *gin.RouterGroup) *gin.RouterGroup {
	v1Group := group.Group("/v1")
	v1Group.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"apiVersion": "v1",
		})
	})
	routerGroupV1Local(config, v1Group)
	routerGroupV1Git(config, v1Group)

	return v1Group
}
