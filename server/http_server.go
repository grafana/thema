package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/grafana/thema/server/storage"
)

type HTTPServer struct {
	port  string
	store storage.Store
}

func Init(port string) error {
	s := HTTPServer{
		port:  port,
		store: storage.NewFileStore("./schemaregistry"),
	}

	return s.Serve()
}

func (s *HTTPServer) Serve() error {
	r := chi.NewRouter()

	r.Route("/schemaregistry", func(r chi.Router) {
		r.Post("/publish-lineage/{registry}", s.PublishLineage)
	})

	return http.ListenAndServe(":"+s.port, r)
}
