// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/security-onion-solutions/securityonion-soc/web"
)

type GridHandler struct {
	server *Server
}

func RegisterGridRoutes(srv *Server, r chi.Router, prefix string) {
	h := &GridHandler{
		server: srv,
	}

	r.Route(prefix, func(r chi.Router) {
		r.Get("/", h.getNodes)
	})
}

func (h *GridHandler) getNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	nodes := h.server.Datastore.GetNodes(ctx)

	web.Respond(w, r, http.StatusOK, nodes)
}
