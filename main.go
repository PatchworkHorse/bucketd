package main

import (
	"context"
	"go-object-api/objectDns"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Default redis options
var redisOptions = &redis.Options{
	Addr:     "redis:6379",
	Password: "",
	DB:       0,
	Protocol: 2,
}

func main() {

	r := gin.Default()

	r.GET("/hello", getHello)
	r.GET("/object/:key", func(c *gin.Context) {
		getCache(c, redisOptions)
	})
	r.POST("/object/:key/:value/:expire", func(c *gin.Context) {
		setCache(c, redisOptions)
	})

	go objectDns.StartDnsListener()
	r.Run()

}

func getHello(ctx *gin.Context) {

	time := time.Now().Format("2006-01-02 15:04:05")
	host, _ := os.Hostname()

	ctx.String(http.StatusOK,
		"Hello! The current system time is %s, your request was handled by %s\n", time, host)
}

// Todo: consolidate Redis options

func getCache(gctx *gin.Context, redisOptions *redis.Options) {
	rdb := redis.NewClient(redisOptions)

	ctx := context.Background()
	key := gctx.Param("key")

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

	gctx.String(http.StatusAccepted, value+"\n")
	gctx.Header("X-Expires-In", ttl.String())
}

func setCache(gctx *gin.Context, redisOptions *redis.Options) {
	rdb := redis.NewClient(redisOptions)

	ctx := context.Background()

	keyParam := gctx.Param("key")
	valueParam := gctx.Param("value")
	expireParam := gctx.Param("expire")

	println(expireParam)
	expire, err := strconv.Atoi(expireParam)

	if err != nil {
		gctx.String(http.StatusBadRequest, "Invalid expire parameter")
		return
	}

	err = rdb.Set(ctx, keyParam, valueParam, time.Duration(expire)*time.Second).Err()

	if err != nil {
		gctx.String(http.StatusInternalServerError, "Failed to set cache")
		return
	}

	gctx.String(http.StatusOK, "Cache set successfully!\n")
}
