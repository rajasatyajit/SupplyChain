package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// adminCreateAccount creates a new account (owner-only)
// Body: { "name": "Acme Inc", "email": "owner@example.com" }
func (h *Handler) adminCreateAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
if err := jsonNewDecoder(r, &body); err != nil {
		h.writeErrorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if body.Name == "" {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "name is required")
		return
	}
	// Insert account
	row := h.db.QueryRow(r.Context(), "INSERT INTO accounts(id,name,email) VALUES (gen_random_uuid(), $1, $2) RETURNING id", body.Name, body.Email)
var id uuid.UUID
	if err := scanRow(row, &id); err != nil {
		h.writeErrorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, map[string]any{"account_id": id.String()})
}

// Helpers to decode JSON and scan pgx row without importing encoder/pgx types here
func jsonNewDecoder(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func scanRow(row interface{}, dest ...any) error {
	if s, ok := row.(interface{ Scan(dest ...any) error }); ok {
		return s.Scan(dest...)
	}
	return fmt.Errorf("invalid row type")
}
