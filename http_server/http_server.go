package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	serverCtx   context.Context
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

func (h *HttpServer) Stop() {
	h.serverCtx.Done()
}

func chainMiddlewares(h http.Handler, middlewares ...interfaces.MiddlewareFn) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- { // reverse order: first added runs first
		h = middlewares[i](h)
	}
	return h
}

func runServer(ctx context.Context, s *http.Server, shutdownTimeout time.Duration) error {
	serverErrCh := make(chan error, 1)

	go func() {
		log.Println("Server starting")
		if err := s.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		return err
	case <-stop:
		log.Println("Shutdown Signal recieved")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		if closErr := s.Close(); closErr != nil {
			return errors.Join(closErr, err)
		}
		return err
	}

	log.Println("server closed")
	return nil
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
		h.serverCtx = context.Background()
	}
	return runServer(h.serverCtx, s, 5*time.Second)
}
