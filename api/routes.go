package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	api := s.router.PathPrefix("/v1/s3").Subrouter()
	api.HandleFunc("/ping", s.PingHandler).Methods(http.MethodGet)
	api.HandleFunc("/version", s.VersionHandler).Methods(http.MethodGet)
	api.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	// buckets handlers
	api.HandleFunc("/{account}/buckets", s.BucketListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets", s.BucketCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketHeadHandler).Methods(http.MethodHead)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/buckets/{bucket}", s.BucketUpdateHandler).Methods(http.MethodPut)

	// bucket users handlers
	api.HandleFunc("/{account}/buckets/{bucket}/users", s.UserListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}/users", s.UserCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/buckets/{bucket}/users/{user}", s.UserShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/buckets/{bucket}/users/{user}", s.UserDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/buckets/{bucket}/users/{user}", s.UserUpdateKeyHandler).Methods(http.MethodPut)

	// websites handlers
	api.HandleFunc("/{account}/websites", s.CreateWebsiteHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/websites/{bucket}", s.BucketHeadHandler).Methods(http.MethodHead)
	api.HandleFunc("/{account}/websites/{website}", s.WebsiteShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/websites/{website}", s.WebsiteDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/websites/{bucket}", s.BucketUpdateHandler).Methods(http.MethodPut)

	// website users handlers
	api.HandleFunc("/{account}/websites/{bucket}/users", s.UserListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/websites/{bucket}/users", s.UserCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/websites/{bucket}/users/{user}", s.UserShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/websites/{bucket}/users/{user}", s.UserDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/websites/{bucket}/users/{user}", s.UserUpdateKeyHandler).Methods(http.MethodPut)
}
