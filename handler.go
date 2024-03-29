package preview

import "net/http"

func (s *Server) handleProxy(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("proxying request", "path", prefix, "url", r.URL)
		r.Host = s.proxyTarget.Host
		s.proxy.ServeHTTP(w, r)
	}
}
