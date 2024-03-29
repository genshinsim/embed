package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/genshinsim/preview"
	"github.com/redis/go-redis/v9"
)

//go:embed dist/*
var content embed.FS

func main() {
	server, err := preview.New(content, redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	},
		preview.WithProxy("/api", "https://gcsim.app/"),
		preview.WithLocalAssets("/api/assets", os.Getenv("ASSETS_DATA_PATH")),
		preview.WithSkipTLSVerify(),
	)
	if err != nil {
		panic(err)
	}
	log.Println("starting img generation listener")
	log.Fatal(http.ListenAndServe(":3001", server.Router))
}
