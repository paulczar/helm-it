//go:build !embed
// +build !embed

package main

import "net/http"

func fileServer() http.Handler {
	return http.FileServer(http.Dir("static"))
}
