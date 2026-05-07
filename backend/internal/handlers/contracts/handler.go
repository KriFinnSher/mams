package contracts

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

type Handler struct {
	services ServiceReader
	proto    ProtoReader
}

func NewHandler(services ServiceReader, proto ProtoReader) *Handler {
	return &Handler{services: services, proto: proto}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if svc.OrganizationID != claims.OrganizationID {
		utils.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	raw, err := h.proto.ReadProjectProto(r.Context(), svc.RepositoryURL, svc.DefaultBranch)
	if err != nil {
		if errors.Is(err, githubclient.ErrProtoNotFound) {
			utils.WriteError(w, http.StatusNotFound, "project.proto is missing")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	parsed, err := protoparser.ParseProjectProto(raw)
	if err != nil {
		if errors.Is(err, protoparser.ErrInvalidProto) {
			utils.WriteError(w, http.StatusBadRequest, "project.proto is invalid")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"service_id":   svc.ID.String(),
		"service_name": parsed.ServiceName,
		"methods":      parsed.Methods,
	})
}
