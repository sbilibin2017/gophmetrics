package http

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

// NewDBPingHandler returns an HTTP handler function that checks db connection
func NewDBPingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
