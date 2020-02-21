package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/go-chi/chi"
)

func NewRouteConfigurer(parent, services *chi.Mux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var config ServiceConfig
		err := json.NewDecoder(r.Body).Decode(&config)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}

		prefix := config.Prefix

		service := chi.NewRouter()
		service.Use(GetMiddleware(config.Middleware...)...)
		services.Mount(prefix, service)

		// Returns error if url is bad
		proxy, _ := NewProxy(ProxyParams{
			Destination:         config.Domain,
			PreserveServiceName: config.PreservePrefix,
		})

		// Recursively add routes with corresponding middleware
		ConfigureRoutes(ConfigureRoutesParams{
			Parent: service,
			Routes: config.Routes,
			Proxy:  proxy,
		})

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
}

type ConfigureRoutesParams struct {
	Parent *chi.Mux
	Routes []ServiceConfigRoute
	Proxy  http.Handler
}

func ConfigureRoutes(params ConfigureRoutesParams) {
	for _, route := range params.Routes {
		middleware := GetMiddleware(route.Middleware...)
		if len(route.Routes) > 0 {
			child := chi.NewRouter()
			// Add middleware here
			child.Use(middleware...)
			params.Parent.Mount(route.Pattern, child)
			ConfigureRoutes(ConfigureRoutesParams{
				Parent: child,
				Routes: route.Routes,
				Proxy:  params.Proxy,
			})
			return
		}
		params.Parent.With(middleware...).Handle(route.Pattern, params.Proxy)
	}
}

type ServiceConfig struct {
	Repo           string               `json:"repo"`
	Team           string               `json:"team"`
	Domain         string               `json:"domain"`
	Prefix         string               `json:"prefix"`
	PreservePrefix bool                 `json:"preservePrefix"`
	Middleware     []string             `json:"middleware"`
	Routes         []ServiceConfigRoute `json:"routes"`
}

type ServiceConfigRoute struct {
	Pattern     string               `json:"pattern"`
	ContentType string               `json:"contentType"`
	Middleware  []string             `json:"middleware"`
	Routes      []ServiceConfigRoute `json:"routes"`
}

// func validate(config *ServiceConfig) error {
// 	if config.Repo == "" {
// 		return false, fmt.Errorf("invalid configuration: %v")
// 	}
// }

// func (config *ServiceConfig) Parse() error {
// 	if config.Repo == "" {
// 		return errors.New("invalid configuration: missing or empty value for field 'repo'")
// 	}
// 	_, err := url.ParseRequestURI(config.Repo)
// 	if err != nil {
// 		return fmt.Errorf("invalid configuration: malformed value for field 'repo': %v", err)
// 	}
// 	if config.Team == "" {
// 		return errors.New("invalid configuration: missing or empty value for field 'team'")
// 	}
// 	if config.Domain == "" {
// 		return errors.New("invalid configuration: missing or empty value for field 'domain'")
// 	}
// 	_, err = url.ParseRequestURI(config.Domain)
// 	if err != nil {
// 		return fmt.Errorf("invalid configuration: malformed value for field 'domain': %v", err)
// 	}
// 	// We'd want to also validate whether this prefix is unique across all services
// 	if config.Prefix == "" || config.Prefix == "/" {
// 		return errors.New("invalid configuration: missing, empty, or malformed value for field 'prefix': prefix must be a unique identifier across all services and cannot be '/'")
// 	}
// 	if config.Pattern == "" {
// 		config.Pattern = "/"
// 	}
// 	for _, route := range config.Routes {
// 		wildcard := false
// 		for _, method := range route.Methods {
// 			if method == "*" && len(route.Methods) > 0 {
// 				wildcard := true
// 			}
// 			// Would we support all possible methods?
// 			if method != "GET" || method != "POST" {
// 				return errors.New("invalid configuration: one or more routes has one or more invalid values for field 'methods': valid values are 'GET' and 'POST'")
// 			}
// 		}

// 		if route.ContentType == "" {
// 			return errors.New("invalid configuration: one or more routes has missing or empty values for field 'contentType'")
// 		}
// 		if route.ContentType != "json" && route.ContentType != "html" {
// 			return errors.New("invalid configuration: one or more routes has an invalid value for field 'contentType': valid values are 'json' and 'html'")
// 		}
// 	}
// }

// // Setup actual routes
// for _, route := range config.Routes {
// 	pattern := route.Pattern
// 	// This is just to return from this handler in the response
// 	patterns = append(patterns, path.Clean("/services/"+pattern))
// 	proxy := &httputil.ReverseProxy{
// 		Director: func(r *http.Request) {
// 			r.URL.Scheme = destination.Scheme
// 			r.URL.Host = destination.Host
// 			start := 3
// 			if config.PreservePrefix {
// 				start = 2
// 			}
// 			// /services/a/somepath -> /a/somepath
// 			// Note empty string @ index 0 due to splitting on "/"
// 			split := strings.Split(r.URL.Path, "/")[start:]
// 			parsed := make([]string, 0, len(split))
// 			for _, str := range split {
// 				parsed = append(parsed, str)
// 			}
// 			r.URL.Path = "/" + path.Join(parsed...)
// 		},
// 	}
// 	service.Handle(pattern, proxy)
// }

// // Be sure to validate this domain within the config validation
// destination, _ := url.ParseRequestURI(config.Domain)

// proxy := NewMockProxy(MockProxyParams{
// 	Destination:         config.Domain,
// 	PreserveServiceName: config.PreservePrefix,
// })

type ProxyParams struct {
	Destination         string
	PreserveServiceName bool
}

// TODO: Add error handler that checks whether the response is json or html.
// A new instance of this handler will belong to each registered service.
func NewProxy(params ProxyParams) (http.Handler, error) {
	destination, err := url.ParseRequestURI(params.Destination)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = destination.Scheme
			r.URL.Host = destination.Host
			r.URL.Path = TruncatePath(r.URL.Path, params.PreserveServiceName)
		},
	}
	return proxy, nil
}

func TruncatePath(p string, preserveServiceName bool) string {
	split := strings.Split(p, "/")
	// Remove the empty string resulting from a leading "/"
	if split[0] == "" {
		split = split[1:]
	}
	if !preserveServiceName {
		split = split[1:]
	}
	parsed := make([]string, 0, len(split))
	for _, str := range split {
		parsed = append(parsed, str)
	}
	return "/" + path.Join(parsed...)
}

var MiddlewareMap = map[string]func(http.Handler) http.Handler{
	"a": MiddlewareA,
	"b": MiddlewareB,
}

func GetMiddleware(keys ...string) []func(http.Handler) http.Handler {
	middleware := make([]func(http.Handler) http.Handler, 0, len(keys))
	for _, key := range keys {
		m, ok := MiddlewareMap[key]
		if ok {
			middleware = append(middleware, m)
		}
	}
	return middleware
}

func MiddlewareA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Middleware A")
		next.ServeHTTP(w, r)
	})
}

func MiddlewareB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Middleware B")
		next.ServeHTTP(w, r)
	})
}
