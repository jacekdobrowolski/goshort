package links

import (
	"log/slog"
	"net/http"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, pgStore *PostgresStore) {
	// mux.HandleFunc("/api/", handleAPI())
	// mux.HandleFunc("/about", handleAbout())
	mux.HandleFunc("/readyz", handleReadyz)
	// mux.HandleFunc("/admin", adminOnly(handleAdminIndex()))
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
