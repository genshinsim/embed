package main

import (
	"crypto/rand"
	"embed"
	"fmt"
	"log"
	"math/big"

	"github.com/caarlos0/env/v10"
	"github.com/genshinsim/preview"
	"github.com/redis/go-redis/v9"
)

//go:embed dist/*
var content embed.FS

type config struct {
	Host        string `env:"HOST"`
	Port        string `env:"PORT" envDefault:"3000"`
	LauncherURL string `env:"LAUNCHER_URL" envDefault:"ws://launcher:7317"`
	AuthKey     string `env:"AUTH_KEY"`
	PreviewURL  string `env:"PREVIEW_URL" envDefault:"http://preview:3000"`
	// proxy is always used
	ProxyTO     string `env:"PROXY_TO" envDefault:"https://gcsim.app"`
	ProxyPrefix string `env:"PROXY_PREFIX" envDefault:"/api"`
	// this is for local image assets
	LocalAssets string `env:"ASSETS_PATH"`
	AssetPrefix string `env:"ASSETS_PREFIX" envDefault:"/api/assets"`
	// redis options
	RedisURL []string `env:"REDIS_URL" envDefault:"redis:6379" envSeparator:","`
	RedisDB  int      `env:"REDIS_DB" envDefault:"0"`
	// timeouts
	GenerateTimeoutInSec int `env:"GENERATE_TIMEOUT_IN_SEC"`
	CacheTTLInSec        int `env:"CACHE_TTL_IN_SEC"`
}

func main() {
	var err error

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	fmt.Printf("%+v\n", cfg)

	if cfg.AuthKey == "" {
		cfg.AuthKey, err = GenerateRandomString(12)
		log.Println("Authkey not set; ssing random string ", cfg.AuthKey)
		panicErr(err)
	}
	log.Println("running with config ", cfg)

	server, err := preview.New(content, redis.UniversalOptions{
		Addrs: cfg.RedisURL,
		DB:    cfg.RedisDB,
	}, cfg.Host+":"+cfg.Port, cfg.LauncherURL, cfg.PreviewURL, cfg.AuthKey)

	panicErr(err)

	err = server.SetOpts(
		preview.WithProxy(cfg.ProxyPrefix, cfg.ProxyTO),
		preview.WithSkipTLSVerify(),
	)
	panicErr(err)

	if cfg.LocalAssets != "" {
		panicErr(server.SetOpts(preview.WithLocalAssets(cfg.AssetPrefix, cfg.LocalAssets)))
	}

	if cfg.GenerateTimeoutInSec > 0 {
		panicErr(server.SetOpts(preview.WithGenerateTimeout(cfg.GenerateTimeoutInSec)))
	}

	if cfg.CacheTTLInSec > 0 {
		panicErr(server.SetOpts(preview.WithCacheTTL(cfg.CacheTTLInSec)))
	}

	log.Println("starting img generation listener")
	log.Fatal(server.Start())
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}
