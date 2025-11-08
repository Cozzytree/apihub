package interfaces

import (
	"net/http"
	"time"
)

type ServerConfig struct {
	Host                 string
	Port                 uint16
	Max_request_size     uint
	Request_timeout_ms   uint64
	Max_header_size      uint
	Rate_limit           bool
	Rate_limit_requests  uint32
	Rate_limit_window_ms time.Duration
}

type HandlerFn func(writer http.ResponseWriter, request *http.Request)

type MiddlewareFn func(next http.Handler) http.Handler

type Server interface {
	Start(config ServerConfig) error
	Stop()
	AddRoute(method string, path string, handler HandlerFn)
	AddMiddleware(middleware MiddlewareFn)
}
