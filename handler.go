package preview

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
)

func (s *Server) handleProxy(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("proxying request", "path", prefix, "url", r.URL)
		r.Host = s.proxyTarget.Host
		s.proxy.ServeHTTP(w, r)
	}
}

var ErrDir = errors.New("path is dir")

func (s *Server) tryRead(requestedPath string, w http.ResponseWriter) error {
	f, err := s.staticFS.Open(path.Join("dist", requestedPath))
	if err != nil {
		return err
	}
	defer f.Close()

	stat, _ := f.Stat()
	if stat.IsDir() {
		return ErrDir
	}

	contentType := mime.TypeByExtension(filepath.Ext(requestedPath))
	w.Header().Set("Content-Type", contentType)
	_, err = io.Copy(w, f)
	return err
}

func (s *Server) notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("received request on not found handler", "path", r.URL)
		err := s.tryRead(r.URL.Path, w)
		if err == nil {
			s.logger.Info("file found; serving", "path", r.URL)
			return
		}
		s.logger.Info("file not found; trying index.html next", "path", r.URL, "err", err)
		err = s.tryRead("index.html", w)
		if err != nil {
			s.logger.Info("error reading index.html", "path", r.URL, "err", err)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
