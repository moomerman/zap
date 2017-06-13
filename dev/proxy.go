package dev

import (
	"fmt"
	"net/http"
)

func proxyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(" [proxyHandler] serve")
	}
}
