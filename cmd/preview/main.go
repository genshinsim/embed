package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/genshinsim/preview"
	"github.com/redis/go-redis/v9"
)

//go:embed dist/*
var content embed.FS

func main() {
	server, err := preview.New(content, redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	},
		preview.WithProxy("/api", "https://gcsim.app/"),
		preview.WithLocalAssets("/api/assets", "/home/srliao/code/assets/assets"),
	)
	if err != nil {
		panic(err)
	}
	log.Println("starting img generation listener")
	log.Fatal(http.ListenAndServe("localhost:3001", server.Router))
}
