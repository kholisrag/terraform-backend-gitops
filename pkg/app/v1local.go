package app

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/kholisrag/terraform-backend-gitops/pkg/encryptions"
	"github.com/kholisrag/terraform-backend-gitops/pkg/lock/redis"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
	"go.uber.org/zap"
)

var (
	Locker *redis.RedisLocker
)

func routerGroupV1Local(config *config.Config, group *gin.RouterGroup) *gin.RouterGroup {
	v1Local := group.Group("/local")
	v1Local.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"backend":    "local",
			"apiVersion": "v1",
		})
	})
	v1Local.POST("/state", applyHandler(config))
	v1Local.GET("/state", getHandler(config))
	v1Local.Handle("LOCK", "/lock", lockHandler(config))
	v1Local.Handle("UNLOCK", "/unlock", unlockHandler())
	return v1Local
}

func applyHandler(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		relativeStatePath := c.Query("state")
		logger.Debugf("applyHandler relativeStatePath: %s", relativeStatePath)
		stateData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("failed to read request body", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(400, err)
		}

		var data interface{}
		err = json.Unmarshal(stateData, &data)
		if err != nil {
			logger.Error("failed to unmarshal request body", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(400, err)
		}

		statePath := filepath.Join(config.Repo.RepoLocal.Path, relativeStatePath)
		dirPath := filepath.Dir(statePath)
		logger.Debugf("applyHandler statePath: %s", statePath)
		logger.Debugf("applyHandler dirPath: %s", dirPath)

		err = os.MkdirAll(dirPath, 0750)
		if err != nil {
			logger.Error("failed to create state directory", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(500, err)
		}
		stateFile, err := os.Create(statePath)
		if err != nil {
			logger.Error("failed to create state file", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(500, err)
		}
		defer stateFile.Close()

		err = encryptions.AgeEncrypt(config.Encryptions.Age.Recipient, string(stateData), stateFile)
		if err != nil {
			logger.Error("failed to write encrypted state file", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(500, err)
		}
		if err != nil {
			logger.Error("failed to write state file", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(500, err)
		}

		c.JSON(200, gin.H{
			"message": "applied successfully",
			"status":  "ok",
			"state":   relativeStatePath,
		})
	}
}

func getHandler(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		relativeStatePath := c.Query("state")
		body, err := io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()
		if err != nil {
			logger.Error("failed to read request body", zap.Error(err))
			//nolint:errcheck
			c.AbortWithError(400, err)
		}
		logger.Debugf("requestbody: %s", body)
		logger.Debugf("getHandler relativeStatePath: %s", relativeStatePath)

		statePath := filepath.Join(config.Repo.RepoLocal.Path, relativeStatePath)
		logger.Debugf("statePath: %s", statePath)

		// Check if file exists or not if not return 404
		// If file exists then decrypt and return the file
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			c.AbortWithStatus(404)
		} else {
			logger.Debugf("file exists")
			stateFile, err := encryptions.AgeDecrypt(config.Encryptions.Age.AgePrivateKeyPath, statePath)
			if err != nil {
				logger.Debugf("get err: %v", err)
				if err.Error() == "failed to open file" {
					c.AbortWithStatusJSON(404, err)
				} else {
					logger.Error("failed to decrypt file", zap.Error(err))
					//nolint:errcheck
					c.AbortWithError(500, err)
					return
				}
			} else {

				c.JSON(200, stateFile)
			}
		}
	}
}

func lockHandler(config *config.Config) gin.HandlerFunc {
	Locker = redis.NewRedisLock(config)
	return func(c *gin.Context) {
		relativeStatePath := c.Query("state")

		lock, err := Locker.GetLock(relativeStatePath)
		if lock == "not_found" {
			c.AbortWithStatusJSON(200, gin.H{
				"message": "lock not found",
				"status":  "not_found",
				"state":   relativeStatePath,
			})
		}
		if err != nil {
			logger.Errorf("failed to get lock: %v", err)
			//nolint:errcheck
			c.AbortWithError(500, err)
		}
		if lock == relativeStatePath {
			c.AbortWithStatusJSON(500, gin.H{
				"message": "lock already exists",
				"status":  "already_exists",
				"state":   relativeStatePath,
			})
		}

		logger.Debug("starting to lock using redsync")

		logger.Debugf("lockHandler relativeStatePath: %s", relativeStatePath)
		//nolint:errcheck
		Locker.Lock(relativeStatePath, false)
	}
}

func unlockHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Debug("processing to unlock with redsync")

		relativeStatePath := c.Query("state")
		logger.Debugf("unlockHandler relativeStatePath: %s", relativeStatePath)
		//nolint:errcheck
		Locker.Unlock(relativeStatePath)
	}
}
