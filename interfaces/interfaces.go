package interfaces

import (
	"net/http"
)

type ServerConfig struct {
	Host                 string
	Port                 uint16
	Max_request_size     uint
	Request_timeout_ms   uint64
	Max_header_size      uint
	Rate_limit_requests  uint32
	Rate_limit_window_ms uint64
}

type HandlerFn func(writer http.ResponseWriter, request *http.Request)

type MiddlewareFn func(next http.Handler) http.Handler

type Server interface {
	Start(config ServerConfig) error
	Stop() error
	AddRoute(method string, path string, handler HandlerFn)
	AddMiddleware(middleware MiddlewareFn)
}
