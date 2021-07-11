package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/georgechang0117/url-shortener/base/cache"
	"github.com/georgechang0117/url-shortener/base/lock"
	"github.com/georgechang0117/url-shortener/core/dao"
	"github.com/georgechang0117/url-shortener/core/urlshortener"
	"github.com/georgechang0117/url-shortener/rest"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	mysqlConnStr  = os.Getenv("MYSQL_CONN_STR")
	mysqlUser     = os.Getenv("MYSQL_USER")
	mysqlPassword = os.Getenv("MYSQL_PASSWORD")

	restHost  = flag.String("rest_host", "", "rest host")
	restPort  = flag.Int("rest_port", 80, "rest port")
	redisAddr = flag.String("redis_addr", "", "redis address")
)

func main() {
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("fail to init zap logger")
	}
	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	if *restHost == "" {
		logger.Sugar().Fatal("redis_host is empty")
	}
	if *redisAddr == "" {
		logger.Sugar().Fatal("redis_addr is empty")
	}
	if mysqlConnStr == "" {
		logger.Sugar().Fatal("mysqlConnStr is empty")
	}

	connStr := mysqlConnStr
	if mysqlUser != "" {
		connStr = fmt.Sprintf("%s:%s@%s", mysqlUser, mysqlPassword, mysqlConnStr)
	}

	mysqlDB, err := gorm.Open(mysql.Open(connStr), &gorm.Config{})
	if err != nil {
		logger.Sugar().Fatalf("fail to connection mysql db, err: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:       *redisAddr,
		PoolSize:   100,
		MaxRetries: 2,
		DB:         0,
	})

	remoteCache := cache.NewRedis(rdb)

	shortLinkDao, err := dao.NewShortLinkDao(mysqlDB)
	if err != nil {
		logger.Sugar().Fatalf("fail to init ShortLinkDao, err: %v", err)
	}
	locker := lock.NewRedis(rdb)
	urlShortener := urlshortener.NewURLShortener(
		locker,
		remoteCache,
		shortLinkDao,
		clock.NewClock(),
	)

	r := rest.NewRest(*restHost, *restPort, urlShortener, clock.NewClock())
	r.Start()
}
