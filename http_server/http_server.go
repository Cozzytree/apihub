package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Cozzytree/apihub/interfaces"
)

type route struct {
	method  string
	path    string
	handler interfaces.HandlerFn
}

type HttpServer struct {
	Config      interfaces.ServerConfig
	Routes      []route
	Middlewares []interfaces.MiddlewareFn
	server      *http.Server
}

func CreateHttpServer() *HttpServer {
	return &HttpServer{}
}

func (h *HttpServer) AddRoute(method string, path string, handler interfaces.HandlerFn) {
	h.Routes = append(h.Routes, route{
		method:  method,
		path:    path,
		handler: handler,
	})
}

func (h *HttpServer) AddMiddleware(middleware interfaces.MiddlewareFn) {
	h.Middlewares = append(h.Middlewares, middleware)
}

func (h *HttpServer) Stop() error {
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	fmt.Println("Shutdown complete.")
	return nil
}

func chainMiddlewares(h http.Handler, middlewares ...interfaces.MiddlewareFn) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- { // reverse order: first added runs first
		h = middlewares[i](h)
	}
	return h
}

func (h *HttpServer) Start(config interfaces.ServerConfig) error {
	mux := &http.ServeMux{}
	fmt.Println("Routes:")
	for _, r := range h.Routes {
		var modifiedPath string
		reqPath := strings.SplitSeq(r.path, "/")
		for p := range reqPath {
			if after, ok := strings.CutPrefix(p, ":"); ok {
				modifiedPath += fmt.Sprintf("{%s}/", after)
			} else {
				modifiedPath += p + "/"
			}
		}

		path := fmt.Sprintf("%s %s", r.method, modifiedPath)
		if newPath, ok := strings.CutSuffix(path, "/"); ok {
			mux.HandleFunc(newPath, r.handler)
			fmt.Println(" ", newPath)
		} else {
			fmt.Println(" ", modifiedPath)
			mux.HandleFunc(modifiedPath, r.handler)
		}
	}
	handler := chainMiddlewares(mux, h.Middlewares...)

	s := &http.Server{
		Handler:        handler,
		Addr:           fmt.Sprintf(":%d", config.Port),
		MaxHeaderBytes: int(config.Max_header_size),
		ReadTimeout:    time.Duration(config.Request_timeout_ms) * time.Millisecond,
		WriteTimeout:   time.Duration(config.Request_timeout_ms) * time.Millisecond,
	}

	if h.server == nil {
		h.server = s
	}
	return s.ListenAndServe()
}
