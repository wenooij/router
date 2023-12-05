package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Router implements a recursive URL router.
//
// Provided handler functions are not HTTP method-specific.
// They should handle all supported HTTP methods and return NotFound for unsupported ones.
type Router struct {
	http.HandlerFunc
	routes   map[string]*Router
	wildcard string    // name of the wildcard route
	once     sync.Once // for routes
}

func (d *Router) initRoutes() {
	d.routes = make(map[string]*Router)
}

// Merge inserts the routes from router into d under the given prefix, squashing any existing routes in d.
func (d *Router) Merge(router *Router) {
	d.once.Do(d.initRoutes)
	d.mergeRec(router)
}

func (d *Router) mergeRec(router *Router) {
	if router.HandlerFunc != nil {
		d.HandlerFunc = router.HandlerFunc
	}
	for elem, route2 := range router.routes {
		route, ok := d.routes[elem]
		if !ok {
			d.routes[elem] = route2
			continue
		}
		route.mergeRec(route2)
	}
}

// Routes returns a list of all routes contained within this router.
func (d *Router) Routes() []string {
	var routes []string
	d.routesRec("", &routes)
	return routes
}

func (d *Router) routesRec(prefix string, routes *[]string) {
	if d.HandlerFunc != nil {
		*routes = append(*routes, prefix)
	}
	for part, route := range d.routes {
		if part == "" {
			part = d.wildcard
		}
		route.routesRec(fmt.Sprint(prefix, "/", part), routes)
	}
}

// Insert the route handler into the router creating new subrouters where needed.
//
// Use a wildcard path element to match any string (e.g. ":user_id").
// The name of the wildcard element will be be retained for documentation purposes.
// Only one wildcard is allowed per Router. Inserting another will overwrite the first.
func (d *Router) Insert(pattern string) *Router {
	d.once.Do(d.initRoutes)
	var sc pathScanner
	sc.Reset(pattern)
	return d.insertRec(&sc)
}

func (d *Router) insertRec(sc *pathScanner) *Router {
	elem := sc.Next()
	if elem == "" {
		return d
	}
	if strings.HasPrefix(elem, ":") {
		d.wildcard = elem
		elem = ""
	}
	route, ok := d.routes[elem]
	if !ok {
		route = new(Router)
		route.once.Do(route.initRoutes)
		d.routes[elem] = route
	}
	return route.insertRec(sc)
}

// InsertFunc inserts the route handler into the router creating subrouters where needed. Set the handler func to f.
//
// Use a wildcard path element to match any string (e.g. ":user_id").
// The name of the wildcard element will be be retained for documentation purposes.
// Only one wildcard is allowed per Router. Inserting another will overwrite the first.
func (d *Router) InsertFunc(pattern string, f http.HandlerFunc) *Router {
	d.once.Do(d.initRoutes)
	var sc pathScanner
	sc.Reset(pattern)
	route := d.insertRec(&sc)
	route.HandlerFunc = f
	return route
}

// ServeHTTP serves the HTTP request through the router.
//
// NotFound is served for requested routes not present in the router.
// Wildcard will be invoked when the path literal fails to match.
func (d *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/") {
		http.NotFound(w, r)
		return
	}
	r.URL.Path = r.URL.Path[1:] // TrimPrefix(... "/")
	r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, "/")
	var sc pathScanner
	sc.Reset(r.URL.Path)
	d.serveRec(&sc, w, r)
}

func (d *Router) serveRec(sc *pathScanner, w http.ResponseWriter, r *http.Request) {
	n := sc.Len()
	elem := sc.Next()
	if elem == "" {
		if d.HandlerFunc == nil {
			http.NotFound(w, r)
			return
		}
		d.HandlerFunc(w, r)
		return
	}
	route, ok := d.routes[elem]
	if !ok {
		route, ok = d.routes[""]
		if !ok {
			http.NotFound(w, r)
			return
		}
	}
	prefix := r.URL.Path[:n-sc.Len()]
	path := strings.TrimPrefix(r.URL.Path, prefix)
	rawPath := strings.TrimPrefix(r.URL.RawPath, prefix)
	if path == "" {
		if route.HandlerFunc != nil {
			route.HandlerFunc(w, r)
			return
		}
	}
	r.URL.Path = path
	r.URL.RawPath = rawPath
	route.serveRec(sc, w, r)
}
