package adapter

import (
	"fmt"
	"net/http"
)

// FullURL reconstructs the full URL from a http request
func FullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprint(r.Method, " ", r.Proto, " ", scheme+"://", r.Host, r.URL)
}
