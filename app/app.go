package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Cozzytree/apihub/config"
	"github.com/Cozzytree/apihub/interfaces"
	"github.com/Cozzytree/apihub/middleware"
)

type RuleHeaderNotMatched struct {
}

func (r RuleHeaderNotMatched) Error() string {
	return "Header not matched"
}

type matcher struct {
}

func (m *matcher) findMatchingRule(request *http.Request, rules []config.Rule) (*config.Rule, error) {
	var errs []error

	for _, r := range rules {
		if ok, err := m.doesRuleMatch(request, &r); ok {
			// Found a matching rule, return immediately
			return &r, nil
		} else if err != nil {
			// Collect errors for debugging/logging
			errs = append(errs, fmt.Errorf("rule %q: %w", r.Request.Path, err))
		}
	}

	// No matching rule found, return all errors
	if len(errs) > 0 {
		combined := "No matching rule found:\n"
		for _, e := range errs {
			combined += "- " + e.Error() + "\n"
		}
		return nil, errors.New(combined)
	}

	// No rules at all
	return nil, errors.New("no rules configured")
}

func (m *matcher) doesRuleMatch(request *http.Request, rule *config.Rule) (bool, error) {
	if !m.matchMethod(request, rule) {
		return false, errors.New("Method not matched")
	}

	if !m.matchPath(request, rule) {
		return false, errors.New("Path not matched")
	}

	if !m.matchHeaders(request, rule) {
		return false, RuleHeaderNotMatched{}
	}

	return true, nil
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

	rule.Request.Params = extractParams(rulePath, reqPath)
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

	a.server.AddMiddleware(middleware.Logger)

	// rate limiter
	if server_config.Rate_limit {
		limiter := middleware.NewRateLimiter(
			server_config.Rate_limit_window_ms,
			uint(server_config.Rate_limit_requests),
		)
		a.server.AddMiddleware(limiter.RateLimitMiddleware)
	}

	return a.server.Start(server_config)
}

func (a *Api) Stop() {
	a.server.Stop()
}

func (a *Api) handleRequest(w http.ResponseWriter, r *http.Request) {
	matching_rule, err := a.matcher.findMatchingRule(r, a.config.Rules)
	if matching_rule == nil {
		fmt.Printf("No matching rule found for Method: %s, Path: %s, err: %v\n", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No matching rule found"))
		return
	}

	if matching_rule.IsMock() {
		a.serveMockRequest(w, matching_rule)
		return
	}

	if matching_rule.IsProxy() {
		a.serveProxyRequest(w, r, matching_rule)
		return
	}
}

func (a *Api) serveMockRequest(w http.ResponseWriter, rule *config.Rule) {
	response := rule.Response
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

func (a *Api) serveProxyRequest(w http.ResponseWriter, r *http.Request, rule *config.Rule) {
	target := rule.Proxy.Url

	// Replace path parameters
	if !rule.IsProxyStatic() && len(rule.Request.Params) > 0 {
		target = substituteProxyParams(target, rule.Request.Params)
	}

	proxyUrl, err := url.Parse(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid proxy target: %v", err), http.StatusBadGateway)
		return
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyUrl.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create proxy request: %v", err), http.StatusInternalServerError)
		return
	}
	proxyReq.Header = make(http.Header)
	for key, values := range r.Header {
		for _, v := range values {
			proxyReq.Header.Add(key, v)
		}
	}

	for key, values := range rule.Proxy.Headers {
		proxyReq.Header.Set(key, values)
	}

	proxyReq.Header.Set("X-Forwarded-By", "apihub")
	client := &http.Client{}

	res, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	w.WriteHeader(res.StatusCode)

	for key, values := range res.Header {
		for _, v := range values {
			r.Header.Set(key, v)
		}
	}

	if _, err := io.Copy(w, res.Body); err != nil {
		fmt.Printf("error copying proxy response: %v\n", err)
	}
}

func substituteProxyParams(template string, params map[string]string) string {
	for key, val := range params {
		placeholder := ":" + key
		template = strings.ReplaceAll(template, placeholder, val)
	}
	return template
}

func extractParams(rulePath, reqPath string) map[string]string {
	ruleParts := strings.Split(strings.Trim(rulePath, "/"), "/")
	reqParts := strings.Split(strings.Trim(reqPath, "/"), "/")
	params := make(map[string]string)

	for i, rulePart := range ruleParts {
		if strings.HasPrefix(rulePart, ":") && i < len(reqParts) {
			params[rulePart[1:]] = reqParts[i]
		}
	}
	return params
}
