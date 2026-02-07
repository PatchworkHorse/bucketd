package bucketHttp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"patchwork.horse/bucketd/config"
)

var rdb *redis.Client

func StartHttpListener(coreCfg *config.CoreConfig, httpConfig *config.HttpConfig, redisConfig *config.RedisConfig) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		DB:       redisConfig.Database,
	})

	r := gin.Default()
	registerRoutes(r, coreCfg, httpConfig)
	r.Run()
}

func registerRoutes(r *gin.Engine, coreConfig *config.CoreConfig, httpConfig *config.HttpConfig) *gin.Engine {

	r.Use(hostnameMiddleware(httpConfig))

	r.GET("/*key", func(c *gin.Context) {
		getCache(c, rdb)
	})

	r.POST("/*key", func(c *gin.Context) {

		if err := validateSetParams(c, coreConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}
		setCache(c, coreConfig, rdb)
	})

	return r

}

func hostnameMiddleware(httpConfig *config.HttpConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Host != httpConfig.Hostname {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Hostname"})
			c.Abort()
			return
		}
		c.Next()
	}
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

func getCache(gctx *gin.Context, rdb *redis.Client) {
	ctx := context.Background()
	key := strings.TrimPrefix(strings.ToLower(gctx.Param("key")), "/")

	value, err := rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		gctx.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	if err != nil {
		gctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cache"})
		return
	}

	ttl, _ := rdb.TTL(ctx, key).Result()

	gctx.JSON(http.StatusOK, gin.H{
		"value":      value,
		"ttl":        int64(ttl.Seconds()),
		"validUntil": time.Now().UTC().Add(ttl),
	})
}

func setCache(gctx *gin.Context, coreConfig *config.CoreConfig, rdb *redis.Client) {
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
		// Allow updates to existing keys even if full
		if rdb.Exists(ctx, keyParam).Val() == 0 {
			gctx.String(http.StatusTooManyRequests, "Max allowed elements has been reached")
			return
		}
	}

	err = rdb.Set(ctx, keyParam, valueParam, time.Duration(expire)*time.Second).Err()

	if err != nil {
		gctx.String(http.StatusInternalServerError, "Failed to set cache")
		return
	}

	gctx.JSON(http.StatusOK, gin.H{
		"message":    "Cache set successfully!",
		"key":        keyParam,
		"ttl":        expire,
		"validUntil": time.Now().UTC().Add(time.Duration(expire) * time.Second),
	})

	gctx.Header("X-BucketD-TTL", strconv.Itoa(expire))
}
