package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
)

// POST /v1/admin/accounts/{account_id}/keys
func (h *Handler) adminCreateKey(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")
	var body struct {
		ClientType string `json:"client_type"`
		Label      string `json:"label"`
		Env        string `json:"env"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.ClientType != "agent" && body.ClientType != "human" {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "client_type must be agent or human")
		return
	}
	repo := auth.NewRepository(h.db)
	raw, id, err := repo.CreateAPIKey(r.Context(), accountID, body.ClientType, body.Label, body.Env)
	if err != nil {
		h.writeErrorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, map[string]any{"api_key": raw, "key_id": id})
}

// GET /v1/admin/accounts/{account_id}/keys (stub)
func (h *Handler) adminListKeys(w http.ResponseWriter, r *http.Request) {
	h.writeErrorResponse(w, r, http.StatusNotImplemented, "list keys not implemented yet")
}

// POST /v1/admin/keys/{key_id}/revoke
func (h *Handler) adminRevokeKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "key_id")
	repo := auth.NewRepository(h.db)
	if err := repo.RevokeAPIKey(r.Context(), keyID); err != nil {
		h.writeErrorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]any{"status": "revoked", "key_id": keyID})
}
