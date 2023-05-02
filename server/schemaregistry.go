package server

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/grafana/thema"
	"github.com/grafana/thema/server/utils"
)

func (s *HTTPServer) PublishLineage(w http.ResponseWriter, req *http.Request) {
	registry := chi.URLParam(req, "registry")
	if registry == "" {
		writeError(w, http.StatusBadRequest, "missing registry name")
		return
	}

	if req.Body == nil {
		writeError(w, http.StatusBadRequest, "missing lineage content")
		return
	}

	defer func() { _ = req.Body.Close() }()

	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lin, err := utils.GetLineageFromBytes(bytes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	oldLineage, err := s.store.GetLineage(registry, lin.Name())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !thema.IsAppendOnly(oldLineage, lin) {
		writeError(w, http.StatusBadRequest, "lineages must be append-only")
		return
	}

	err = s.store.StoreLineage(registry, lin.Name(), bytes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
}
