package redis

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
)

const (
	redisLockKey = "terraform-backend-gitops"
)

type RedisLocker struct {
	pool     *redis.Pool
	rsClient *redsync.Redsync
}

func redigoNewPool(config *config.Config) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.Dial("tcp", config.Redis.Addresses[0]) },
	}
}

func NewRedisLock(config *config.Config) *RedisLocker {
	pool := redigoNewPool(config)
	rsPool := redigo.NewPool(pool)

	return &RedisLocker{
		pool:     pool,
		rsClient: redsync.New(rsPool),
	}
}

func (l *RedisLocker) Unlock(path string) (unlocked bool, err error) {
	mutex := l.rsClient.NewMutex(
		redisLockKey,
		redsync.WithExpiry(24*time.Hour),
		redsync.WithTries(1),
		redsync.WithGenValueFunc(func() (string, error) {
			return uuid.New().String(), nil
		}))
	if err := mutex.Lock(); err != nil {
		logger.Errorf("failed to lock redsync mutex: %v", err)
		return false, err
	}

	defer func() {
		if _, mutexErr := mutex.Unlock(); mutexErr != nil {
			logger.Errorf("failed to unlock redsync mutex: %v", mutexErr)
			if err != nil {
				err = mutexErr
			}
		}
	}()

	lock, err := l.getLock(path)
	if err != nil {
		logger.Debugf("redis unlocked status false: %v", lock)
		logger.Errorf("failed to get lock: %v", err)
		return false, nil
	}

	if lock != path {
		return false, nil
	}

	if err := l.deleteLock(path); err != nil {
		logger.Errorf("failed to delete lock: %v", err)
		return false, err
	}

	return true, nil
}

func (l *RedisLocker) Lock(path string, unlock bool) (redisKeyLocked bool, err error) {
	mutex := l.rsClient.NewMutex(
		redisLockKey,
		redsync.WithExpiry(24*time.Hour),
		redsync.WithTries(1),
		redsync.WithGenValueFunc(func() (string, error) {
			return uuid.New().String(), nil
		}))
	if err := mutex.Lock(); err != nil {
		logger.Errorf("failed to lock redsync mutex: %v", err)
		return false, err
	}

	defer func() {
		if _, mutexErr := mutex.Unlock(); mutexErr != nil {
			logger.Errorf("failed to unlock redsync mutex: %v", mutexErr)
			if err != nil {
				err = mutexErr
			}
		}
	}()

	lock, err := l.getLock(path)
	if lock == "not_found" {
		logger.Warnf("lock not found: %v", path)

		if err := l.setLock(path, lock); err != nil {
			return false, err
		}

		return true, err
	}

	if lock == path {
		return true, nil
	}

	return false, err
}

func (l *RedisLocker) GetLock(path string) (lock string, err error) {
	mutex := l.rsClient.NewMutex(
		redisLockKey,
		redsync.WithExpiry(24*time.Hour),
		redsync.WithTries(1),
		redsync.WithGenValueFunc(func() (string, error) {
			return uuid.New().String(), nil
		}))
	if err := mutex.Lock(); err != nil {
		logger.Errorf("failed to lock: %v", err)
		return "", err
	}

	defer func() {
		if _, mutexErr := mutex.Unlock(); mutexErr != nil {
			logger.Errorf("failed to unlock: %v", mutexErr)
			if err != nil {
				err = mutexErr
			}
		}
	}()

	return l.getLock(path)
}

func (l *RedisLocker) getLock(path string) (string, error) {
	ctx := context.Background()

	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		logger.Errorf("failed to get redis connection: %v", err)
		return "", err
	}
	defer conn.Close()

	value, err := redis.String(conn.Do("GET", path))
	if err != nil {
		logger.Debugf("failed to get redis key: %v", err)
		return "not_found", nil
	}
	return value, nil
}

func (l *RedisLocker) deleteLock(path string) error {
	ctx := context.Background()

	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		logger.Errorf("failed to get redis connection: %v", err)
		return err
	}
	defer conn.Close()

	count, err := redis.Int(conn.Do("DEL", path))
	if err != nil {
		logger.Errorf("failed to delete redis key: %v", err)
		return err
	}

	if count != 1 {
		return fmt.Errorf("delete %v redis key while unlocking id %v", count, path)
	}

	return nil
}

func (l *RedisLocker) setLock(path, lock string) error {
	ctx := context.Background()

	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		logger.Errorf("failed to get redis connection: %v", err)
		return err
	}
	defer conn.Close()

	lockValue := base64.StdEncoding.EncodeToString([]byte(lock))

	resp, err := redis.String(conn.Do("SET", path, lockValue, "NX", "PX", int(24*time.Hour/time.Millisecond)))
	if err != nil {
		logger.Errorf("failed to set redis key: %v", err)
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("failed to set redis key: %v", resp)
	}

	return nil
}
