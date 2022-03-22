package rpczapp

import (
	"github.com/go-redis/redis/v8"
	//log "github.com/sirupsen/logrus"
	"net/url"
	"strconv"
)

func redisOptions(redisUrl string) (*redis.Options, error) {
	u, err := url.Parse(redisUrl)
	if err != nil {
		return nil, err
	}
	sdb := u.Path[1:]
	db := 0
	if sdb != "" {
		db, err = strconv.Atoi(sdb)
		if err != nil {
			return nil, err
		}
	}
	pwd, ok := u.User.Password()
	if !ok {
		pwd = ""
	}
	opt := &redis.Options{
		Addr:     u.Host,
		Password: pwd,
		DB:       db,
	}
	return opt, nil
}

func NewRedisClient(redisUrl string) (*redis.Client, error) {
	opts, err := redisOptions(redisUrl)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}
