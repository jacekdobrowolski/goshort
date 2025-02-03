package links

import (
	"bytes"
	"fmt"
	//nolint: gosec // md5 is not used here for any security critical hashing
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/jacekdobrowolski/goshort/pkg/base62"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, store Store) {
	mux.HandleFunc("GET /readyz", handleReadyz)
	mux.HandleFunc("GET /api/v1/links/{short}", handlerGetLink(logger, store))
	mux.HandleFunc("POST /api/v1/links", handlerCreateLink(logger, store))
	mux.HandleFunc("GET /{short}", handlerRedirect(logger, store))
}

func handleReadyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type Link struct {
	Short    string `json:"short"`
	Original string `json:"original"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(v)

	return fmt.Errorf("error encoding json: %w", err)
}

func handlerCreateLink(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType, ok := r.Header["Content-Type"]
		if !ok {
			logger.Debug("no Content-Type header")

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		requestBody := struct {
			URL string `json:"url"`
		}{}

		switch contentType[0] {
		case "application/json":
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				logger.Debug("error parsing json request body no url field")

				w.WriteHeader(http.StatusBadRequest)

				return
			}

			if len(requestBody.URL) == 0 {
				logger.Debug("error parsing json request body empty url")

				w.WriteHeader(http.StatusBadRequest)

				return
			}

			if _, err := url.ParseRequestURI(requestBody.URL); err != nil {
				logger.Debug("error request body contains invalid url")

				w.WriteHeader(http.StatusBadRequest)

				return
			}
		default:
			logger.Debug("unexpected content type", "type", contentType)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		//nolint: gosec // md5 is fine for link shortening but probably slower than it could be
		h := md5.New()

		_, err := io.WriteString(h, requestBody.URL)
		if err != nil {
			logger.Error("error writing to hash", "err", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		hash := h.Sum(nil)
		reader := bytes.NewReader(hash[:4])

		var x uint32
		if err := binary.Read(reader, binary.LittleEndian, &x); err != nil {
			logger.Error("error reading bytes", "err", err)
		}

		short := base62.Encode(uint64(x))

		logger.Debug("hash generated", "url", requestBody.URL, "hash", short)

		if err := store.addLink(short, requestBody.URL); err != nil {
			logger.Error("error adding row into db", "err", err)
		}

		link := Link{
			Short:    path.Join(r.Host, short),
			Original: requestBody.URL,
		}

		err = WriteJSON(w, http.StatusCreated, link)
		if err != nil {
			logger.Error("error writing JSON response", "err", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func handlerGetLink(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		original, err := store.getOriginal(r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))

			w.WriteHeader(http.StatusNotFound)

			return
		}

		link := Link{
			Short:    path.Join(r.Host, r.PathValue("short")),
			Original: *original,
		}

		err = WriteJSON(w, http.StatusOK, link)
		if err != nil {
			logger.Error("error writing JSON response", "err", err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func handlerRedirect(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		original, err := store.getOriginal(r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))

			w.WriteHeader(http.StatusNotFound)

			return
		}

		http.Redirect(w, r, *original, http.StatusTemporaryRedirect)
	}
}
