package s3api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	api := s.router.PathPrefix("/v1/s3").Subrouter()
	api.HandleFunc("/ping", s.PingHandler)
	api.HandleFunc("/version", s.VersionHandler)
	api.Handle("/metrics", promhttp.Handler())

	// Buckets handlers
	api.HandleFunc("/{account}/buckets", s.BucketListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets", s.BucketCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketHeadHandler).Methods(http.MethodHead)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketUpdateHandler).Methods(http.MethodPut)

	// api.HandleFunc("/{account}/buckets/{bucket}/objects", s.ObjectCountHandler).Methods(http.MethodHead)
	api.HandleFunc("/{account}/buckets/{bucket}/users", s.UserListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}/users", s.UserCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/buckets/{bucket}/users/{user}", s.UserDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/buckets/{bucket}/users/{user}", s.UserUpdateKeyHandler).Methods(http.MethodPut)
}
