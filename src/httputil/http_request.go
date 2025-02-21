package httputil

import (
	"net/http"
	"strings"
)

// GetUrlComponents parses request URL into its "/" delimited components
// Copied from OpenLambda src
func GetUrlComponents(r *http.Request) []string {
	path := r.URL.Path
	// trim prefix
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// trim trailing "/"
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return strings.Split(path, "/")
}

func Get2ndPathSegment(r *http.Request, firstSegment string) string {
	components := GetUrlComponents(r)

	if len(components) != 2 {
		return ""
	}

	if components[0] != firstSegment {
		return ""
	}

	return components[1]
}
