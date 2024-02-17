package app

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"go.uber.org/zap"
)

func routerGroupLocal(logger *zap.Logger, config *config.Config, router *gin.Engine) *gin.RouterGroup {
	v1 := router.Group("/local")
	routerGroupLocalV1(logger, config, router)
	return v1
}

func routerGroupLocalV1(logger *zap.Logger, config *config.Config, router *gin.Engine) *gin.RouterGroup {
	v1 := router.Group("/v1")
	v1.POST("/apply", ginApplyHandler(logger, config))
	return v1
}

func ginApplyHandler(logger *zap.Logger, config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		relativeStatePath := c.Query("state")
		logger.Sugar().Debugf("relativeStatePath: %s", relativeStatePath)
		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("failed to read request body", zap.Error(err))
			c.AbortWithError(400, err)
		}

		var data interface{}
		err = json.Unmarshal(jsonData, &data)
		if err != nil {
			logger.Error("failed to unmarshal request body", zap.Error(err))
			c.AbortWithError(400, err)
		}

		statePath := filepath.Join(config.Repo.RepoLocal.Path, relativeStatePath)

		dirPath := filepath.Dir(statePath)
		err = os.MkdirAll(dirPath, 0750)
		if err != nil {
			logger.Error("failed to create state directory", zap.Error(err))
			c.AbortWithError(500, err)
		}

		err = os.WriteFile(statePath, jsonData, 0750)
		if err != nil {
			logger.Error("failed to write state file", zap.Error(err))
			c.AbortWithError(500, err)
		}

		c.JSON(200, gin.H{
			"message": "apply",
			"status":  "ok",
			"state":   relativeStatePath,
		})
	}
}
