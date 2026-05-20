package server

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/persistence"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func HandleGraphQL(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Very basic mock resolver logic
		var data interface{}
		if db != nil {
			var calls []persistence.Call
			db.SelectContext(r.Context(), &calls, "SELECT * FROM calls ORDER BY started_at DESC LIMIT 10")
			data = map[string]interface{}{"calls": calls}
		}

		response := map[string]interface{}{
			"data": data,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
