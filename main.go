package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// var store = make(map[string]string)

func main() {
	r := gin.Default()

	r.GET("/hello", getHello)
	r.GET("/object/:key", getCache)
	r.POST("/object/:key/:value/:expire", setCache)

	r.Run()
}

func getHello(ctx *gin.Context) {

	time := time.Now().Format("2006-01-02 15:04:05")
	host, _ := os.Hostname()

	ctx.String(http.StatusOK,
		"Hello! The current system time is %s, your response was handled by %s\n", time, host)
}

// Todo: consolidate Redis options

func getCache(gctx *gin.Context) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})

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

	gctx.Header("X-Expires-In", ttl.String())
	gctx.String(http.StatusAccepted, value)
}

func setCache(gctx *gin.Context) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})

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

	gctx.String(http.StatusOK, "Cache set successfully")
}
