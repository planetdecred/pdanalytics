package web

import (
	"net/http"

	"github.com/go-chi/chi"
)

type middleware func(next http.Handler) http.Handler

type method string

const (
	GET     method = "get"
	POST    method = "post"
	PUT     method = "put"
	PATCH   method = "patch"
	DELETE  method = "delete"
	OPTIONS method = "options"
)

type route struct {
	Path       string
	Mathod     method
	Handler    http.HandlerFunc
	Middleware []func(next http.Handler) http.Handler
}

func (s *Server) AddRoute(path string, method method, handlerFunc http.HandlerFunc, middleware ...func(next http.Handler) http.Handler) {
	if _, found := s.routes[path]; found {
		panic("duplicate route")
	}
	s.routes[path] = route{
		path, method, handlerFunc, middleware,
	}
}

// route group
type routeGroup struct {
	Prefix     string
	Middleware []func(next http.Handler) http.Handler
	Routes     map[string]route
}

func (s *routeGroup) Add(path string, method method, handlerFunc http.HandlerFunc, middleware ...func(next http.Handler) http.Handler) {
	if _, found := s.Routes[path]; found {
		panic("duplicate route")
	}
	s.Routes[path] = route{
		path, method, handlerFunc, middleware,
	}
}

func (s *Server) RouteGroup(prefix string, a func(r routeGroup) routeGroup, middleware ...func(next http.Handler) http.Handler) {
	routeGroup := routeGroup{
		prefix, middleware, map[string]route{},
	}
	routeGroup = a(routeGroup)
	s.routeGroups = append(s.routeGroups, routeGroup)
}

func (s *Server) BuildRoute() {
	var hasRootPage bool
	var defaultHandler route

	for path, r := range s.routes {
		switch r.Mathod {
		case GET:
			s.webMux.With(r.Middleware...).Get(path, r.Handler)
		case POST:
			s.webMux.With(r.Middleware...).Post(path, r.Handler)
		case PUT:
			s.webMux.With(r.Middleware...).Put(path, r.Handler)
		case PATCH:
			s.webMux.With(r.Middleware...).Patch(path, r.Handler)
		case DELETE:
			s.webMux.With(r.Middleware...).Delete(path, r.Handler)
		case OPTIONS:
			s.webMux.With(r.Middleware...).Options(path, r.Handler)
		}

		if defaultHandler.Handler == nil {
			defaultHandler = r
		}
		if path == "/" {
			hasRootPage = true
		}
	}

	for _, g := range s.routeGroups {
		routes := g.Routes
		s.webMux.With(g.Middleware...).Group(func(router chi.Router) {
			for path, r := range routes {
				switch r.Mathod {
				case GET:
					router.With(r.Middleware...).Get(path, r.Handler)
				case POST:
					router.With(r.Middleware...).Post(path, r.Handler)
				case PUT:
					router.With(r.Middleware...).Put(path, r.Handler)
				case PATCH:
					router.With(r.Middleware...).Patch(path, r.Handler)
				case DELETE:
					router.With(r.Middleware...).Delete(path, r.Handler)
				case OPTIONS:
					router.With(r.Middleware...).Options(path, r.Handler)
				}

				if defaultHandler.Handler == nil {
					defaultHandler = r
				}
				if path == "/" {
					hasRootPage = true
				}
			}
		})
	}

	if !hasRootPage {
		r := defaultHandler
		r.Path = "/"
		switch r.Mathod {
		case GET:
			s.webMux.With(r.Middleware...).Get(r.Path, r.Handler)
		case POST:
			s.webMux.With(r.Middleware...).Post(r.Path, r.Handler)
		case PUT:
			s.webMux.With(r.Middleware...).Put(r.Path, r.Handler)
		case PATCH:
			s.webMux.With(r.Middleware...).Patch(r.Path, r.Handler)
		case DELETE:
			s.webMux.With(r.Middleware...).Delete(r.Path, r.Handler)
		case OPTIONS:
			s.webMux.With(r.Middleware...).Options(r.Path, r.Handler)
		}
	}
}
