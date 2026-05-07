package services

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/githubclient"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/protoparser"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

func (h *Handler) GetContracts(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	svc, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrServiceNotFound) {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		h.log.ErrorCtx(r.Context(), "get service before contracts failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	if h.proto == nil {
		utils.WriteError(w, http.StatusInternalServerError, "contracts reader is not configured")
		return
	}
	raw, err := h.proto.ReadProjectProto(r.Context(), svc.RepositoryURL, svc.DefaultBranch)
	if err != nil {
		if errors.Is(err, githubclient.ErrProtoNotFound) {
			utils.WriteError(w, http.StatusNotFound, "project.proto is missing")
			return
		}
		h.log.ErrorCtx(r.Context(), "read project.proto failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	parsed, err := protoparser.ParseProjectProto(raw)
	if err != nil {
		if errors.Is(err, protoparser.ErrInvalidProto) {
			utils.WriteError(w, http.StatusBadRequest, "project.proto is invalid")
			return
		}
		h.log.ErrorCtx(r.Context(), "parse project.proto failed", "err", err, "service_id", id)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"service_id":    svc.ID.String(),
		"service_name":  parsed.ServiceName,
		"methods":       parsed.Methods,
	})
}

