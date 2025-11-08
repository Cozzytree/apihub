package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Cozzytree/apishop/config"
	"github.com/Cozzytree/apishop/interfaces"
)

type matcher struct {
}

func (m *matcher) findMatchingRule(request *http.Request, rule []config.Rule) *config.Rule {
	for _, r := range rule {
		if m.doesRuleMatch(request, &r) {
			return &r
		}
	}

	return nil
}

func (m *matcher) doesRuleMatch(request *http.Request, rule *config.Rule) bool {
	if !m.matchMethod(request, rule) {
		return false
	}

	if !m.matchPath(request, rule) {
		return false
	}

	if !m.matchHeaders(request, rule) {
		return false
	}

	return true
}

func (m matcher) matchMethod(request *http.Request, rule *config.Rule) bool {
	if rule.Request.Method == request.Method {
		return true
	}

	return false
}

func (m matcher) matchPath(request *http.Request, rule *config.Rule) bool {
	reqPath := strings.TrimSuffix(request.URL.Path, "/")
	rulePath := strings.TrimSuffix(rule.Request.Path, "/")

	reqParts := strings.Split(reqPath, "/")
	ruleParts := strings.Split(rulePath, "/")

	if len(reqParts) != len(ruleParts) {
		return false
	}

	for i := range ruleParts {
		rp := ruleParts[i]
		rq := reqParts[i]

		if strings.HasPrefix(rp, ":") {
			continue
		}

		if rq != rp {
			return false
		}
	}

	return true
}

func (m matcher) matchHeaders(r *http.Request, rule *config.Rule) bool {
	for key, rule_header := range rule.Request.Headers {
		h := r.Header.Get(key)
		if h == "" {
			return false
		}
		if rule_header != h {
			return false
		}
	}
	return true
}

type Api struct {
	server  interfaces.Server
	config  config.Config
	matcher *matcher
}

func Init(srv interfaces.Server, app_config config.Config) Api {
	return Api{
		server:  srv,
		config:  app_config,
		matcher: &matcher{},
	}
}

func (a *Api) Start(server_config interfaces.ServerConfig) error {
	for _, rule := range a.config.Rules {
		a.server.AddRoute(rule.Request.Method, rule.Request.Path, a.handleRequest)
	}
	// a.server.AddRoute(http.MethodGet, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.POST, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.DELETE, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.PATCH, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.PUT, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.HEAD, "/*", a.handleRequest)
	// a.server.AddRoute(interfaces.OPTIONS, "/*", a.handleRequest)

	a.server.AddMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s\n", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	return a.server.Start(server_config)
}

func (a *Api) handleRequest(w http.ResponseWriter, r *http.Request) {
	matching_rule := a.matcher.findMatchingRule(r, a.config.Rules)
	if matching_rule == nil {
		fmt.Printf("No matching rule found for Method: %s, Path: %s\n", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No matching rule found"))
		return
	}

	if matching_rule.IsMock() {
		a.serveMockRequest(w, matching_rule)
		return
	}

	if matching_rule.IsProxy() {
		a.serveProxyRequest(r, matching_rule)
		return
	}
}

func (a *Api) serveMockRequest(w http.ResponseWriter, rule *config.Rule) {
	response := rule.Response
	fmt.Printf("Serving mock response: %d\n", response.Status)
	w.WriteHeader(int(response.Status))
	for key, val := range response.Headers {
		if strVal, ok := val.(string); ok {
			w.Header().Set(key, strVal)
		} else {
			// fallback or error handling
			w.Header().Set(key, fmt.Sprintf("%v", val))
		}
	}
	w.Write([]byte(response.Body))
}

func (a *Api) serveProxyRequest(r *http.Request, rule *config.Rule) {
	client := http.Client{}
	switch r.Method {
	case http.MethodGet:
		client.Get(rule.Proxy.Url)
	}
}
