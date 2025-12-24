package bucketHttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"patchwork.horse/bucketd/config"
)

// Patchwork todo:
// - Respect core config limits (max key length, max value length, max ttl)
// - Respect hostname in HttpConfig (404 if not matching)

var redisOptions *redis.Options

func StartHttpListener(coreCfg *config.CoreConfig, httpConfig *config.HttpConfig, redisConfig *config.RedisConfig) {

	redisOptions = &redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
	}

	r := gin.Default()
	registerRoutes(r, coreCfg, httpConfig)
	r.Run()
}

func registerRoutes(r *gin.Engine, coreConfig *config.CoreConfig, httpConfig *config.HttpConfig) *gin.Engine {

	r.GET("/*key", func(c *gin.Context) {
		validateHostname(c, httpConfig)
		getCache(c, redisOptions)
	})

	r.POST("/*key", func(c *gin.Context) {
		validateHostname(c, httpConfig)
		if err := validateSetParams(c, coreConfig); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		setCache(c, coreConfig, redisOptions)
	})

	return r

}

func validateHostname(gctx *gin.Context, httpConfig *config.HttpConfig) error {
	if gctx.Request.Host == httpConfig.Hostname {
		return nil
	}

	return errors.New("Invalid hostname")
}

func validateSetParams(gctx *gin.Context, coreConfig *config.CoreConfig) error {
	key := strings.TrimPrefix(strings.ToLower(gctx.Param("key")), "/")
	value := gctx.Query("value")

	if coreConfig.MaxKeyLength > 0 && len(key) > coreConfig.MaxKeyLength {
		return fmt.Errorf("key length exceeds maximum allowed (%d)", coreConfig.MaxKeyLength)
	}

	if coreConfig.MaxValueLength > 0 && len(value) > coreConfig.MaxValueLength {
		return fmt.Errorf("value length exceeds maximum allowed (%d)", coreConfig.MaxValueLength)
	}

	expireParam, err := strconv.Atoi(gctx.Query("expire"))
	if err != nil {
		return fmt.Errorf("expire parameter must be a valid integer")
	}

	if expireParam < 0 {
		return fmt.Errorf("expire parameter must be non-negative")
	}

	if coreConfig.MaxTtl > 0 && expireParam > coreConfig.MaxTtl {
		return fmt.Errorf("expire time exceeds maximum allowed (%d)", coreConfig.MaxTtl)
	}

	return nil
}

func getCache(gctx *gin.Context, redisOptions *redis.Options) {
	rdb := redis.NewClient(redisOptions)

	ctx := context.Background()
	key := strings.ToLower(gctx.Param("key"))

	pipe := rdb.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)

	if err != nil && err != redis.Nil {
		gctx.String(http.StatusInternalServerError, "Failed to get cache")
		return
	}

	value, _ := getCmd.Result()
	ttl, _ := ttlCmd.Result()

	gctx.String(http.StatusAccepted, value)
	gctx.Header("X-Expires-In", ttl.String())
}

func setCache(gctx *gin.Context, coreConfig *config.CoreConfig, redisOptions *redis.Options) {
	rdb := redis.NewClient(redisOptions)

	ctx := context.Background()

	keyParam := strings.TrimPrefix(strings.ToLower(gctx.Param("key")), "/")
	valueParam := gctx.Query("value")
	expireParam := gctx.Query("expire")

	expire, err := strconv.Atoi(expireParam)

	if err != nil {
		gctx.String(http.StatusBadRequest, "Invalid expire parameter")
		return
	}

	if count := rdb.DBSize(ctx).Val(); count >= int64(coreConfig.MaxElements) {
		gctx.String(http.StatusTooManyRequests, "Max allowed elements has been reached")
	}

	err = rdb.Set(ctx, keyParam, valueParam, time.Duration(expire)*time.Second).Err()

	if err != nil {
		gctx.String(http.StatusInternalServerError, "Failed to set cache")
		return
	}

	gctx.String(http.StatusOK, "Cache set successfully!")
}
