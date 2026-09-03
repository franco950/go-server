package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type writewrapper struct {
	http.ResponseWriter
	status int
}

func (rw *writerwrapper) WriteHeader2(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			return
		}
		start := time.Now()
		rw := &writewrapper{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s,%s,%d,%v", r.Method, r.URL.Path, rw.status, time.Since(start))

	})
}
func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			return
		}
		if r.Method == "POST" {

			key := r.Header.Get("Api-key")
			apikey := os.Getenv("API_KEY")
			w.Header().Set("Content-Type", "application/json")
			if apikey == "" || apikey != key {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorised"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func Timekeeper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		new := r.WithContext(ctx)
		next.ServeHTTP(w, new)
	})
}
