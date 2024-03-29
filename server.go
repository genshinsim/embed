package preview

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/redis/go-redis/v9"
)

type serverCfg func(s *Server) error

type Server struct {
	logger *slog.Logger
	Router *chi.Mux
	rdb    *redis.Client
	work   chan string

	// static asset; mandatory to serve from
	staticFS embed.FS

	// additional local assets
	useLocalAssets bool
	assetsPrefix   string // cannot be blank
	assetsDir      string

	// proxy requests
	useProxy     bool
	proxyPrefix  string
	skipInsecure bool

	// for proxying api requests
	proxy       *httputil.ReverseProxy
	proxyTarget *url.URL
}

func New(fs embed.FS, connOpt redis.Options, opts ...serverCfg) (*Server, error) {
	s := &Server{
		staticFS: fs,
		work:     make(chan string),
	}
	s.Router = chi.NewRouter()
	for _, f := range opts {
		err := f(s)
		if err != nil {
			return nil, err
		}
	}
	if s.logger == nil {
		s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	err := s.routes()
	if err != nil {
		return nil, err
	}
	s.rdb = redis.NewClient(&connOpt)
	_, err = s.rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	go s.listen()

	return s, nil
}

func WithLogger(logger *slog.Logger) serverCfg {
	return func(s *Server) error {
		s.logger = logger
		return nil
	}
}

func WithLocalAssets(prefix, dir string) serverCfg {
	return func(s *Server) error {
		s.useLocalAssets = true
		s.assetsPrefix = prefix
		s.assetsDir = dir
		return nil
	}
}

func WithProxy(prefix, target string) serverCfg {
	return func(s *Server) error {
		s.useProxy = true
		s.proxyPrefix = prefix
		host, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("error parsing target %v: %w", target, err)
		}
		s.proxyTarget = host
		return nil
	}
}

func WithSkipTLSVerify() serverCfg {
	return func(s *Server) error {
		s.skipInsecure = true
		return nil
	}
}

func (s *Server) routes() error {
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RequestID)

	if s.useLocalAssets {
		localAssetsFS := http.FileServer(http.Dir(s.assetsDir))
		s.Router.Handle(fmt.Sprintf("%v/*", s.assetsPrefix), http.StripPrefix(s.assetsPrefix+"/", localAssetsFS))
	}

	if s.useProxy {
		path := strings.TrimSuffix(s.proxyPrefix, "/")
		s.Router.Handle(path+"/*", s.handleProxy(path))

		s.proxy = httputil.NewSingleHostReverseProxy(s.proxyTarget)
		if s.skipInsecure {
			s.proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}
	}

	s.Router.Handle("/generate/db/{id}", s.handleImageRequest("db"))
	s.Router.Handle("/generate/sh/{id}", s.handleImageRequest("sh"))

	s.Router.NotFound(s.notFoundHandler())

	return nil
}
