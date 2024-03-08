package links

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path"

	"github.com/jacekdobrowolski/goshort/pkg/base62"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, store Store) {
	mux.HandleFunc("GET /readyz", handleReadyz)
	mux.HandleFunc("GET /api/v1/links/{short}", handlerGetLink(logger, store))
	mux.HandleFunc("POST /api/v1/links", handlerCreateLink(logger, store))
	mux.HandleFunc("GET /{short}", handlerRedirect(logger, store))
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

type Link struct {
	Short    string `json:"short"`
	Original string `json:"original"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func handlerCreateLink(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType, ok := r.Header["Content-Type"]
		if !ok {
			logger.Debug("no Content-Type header")
			w.WriteHeader(400)
			return
		}
		requestBody := struct {
			Url string `json:"url"`
		}{}
		switch contentType[0] {
		case "application/json":
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				logger.Debug("error parsing json request body no url field")
				w.WriteHeader(400)
				return
			}
			if len(requestBody.Url) == 0 {
				logger.Debug("error parsing json request body empty url")
				w.WriteHeader(400)
				return
			}
		default:
			logger.Debug("unexpected content type", "type", contentType)
			w.WriteHeader(400)
			return
		}

		h := md5.New()
		io.WriteString(h, requestBody.Url)
		hash := h.Sum(nil)
		reader := bytes.NewReader(hash[:4])
		var x uint32
		if err := binary.Read(reader, binary.LittleEndian, &x); err != nil {
			logger.Error("error reading bytes", "err", err)
		}
		short := base62.Encode(uint64(x))
		logger.Debug("hash generated", "url", requestBody.Url, "hash", short)
		if err := store.addLink(short, requestBody.Url); err != nil {
			logger.Error("error adding row into db", "err", err)
		}
		link := Link{
			Short:    path.Join(r.Host, short),
			Original: requestBody.Url,
		}
		WriteJSON(w, http.StatusOK, link)
	}
}

func handlerGetLink(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		original, err := store.getOriginal(r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))
			w.WriteHeader(404)
			return
		}
		link := Link{
			Short:    path.Join(r.Host, r.PathValue("short")),
			Original: *original,
		}
		WriteJSON(w, http.StatusOK, link)
	}
}

func handlerRedirect(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		original, err := store.getOriginal(r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))
			w.WriteHeader(404)
			return
		}
		http.Redirect(w, r, *original, http.StatusSeeOther)
	}
}
