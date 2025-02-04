package links

import (
	"bytes"
	"context"
	"crypto/md5" //nolint: gosec // md5 used in non security context
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/jacekdobrowolski/goshort/pkg/base62"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var errMissingURLField = errors.New("missing URL field")

func addRoutes(mux *http.ServeMux, logger *slog.Logger, store Store) {
	mux.HandleFunc("GET /readyz", handleReadyz)
	mux.HandleFunc("GET /api/v1/links/{short}", HandlerGetLink(logger, store))
	mux.HandleFunc("POST /api/v1/links", HandlerCreateLink(logger, store))
	mux.HandleFunc("GET /{short}", HandlerRedirect(logger, store))
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
	if err != nil {
		return fmt.Errorf("error encoding json: %w", err)
	}

	return nil
}

func HandlerCreateLink(logger *slog.Logger, store Store) http.HandlerFunc {
	tracer := otel.Tracer("handlercreatelink")

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "handlercreatelink")
		defer span.End()

		logger = logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
		)

		contentType, ok := r.Header["Content-Type"]
		if !ok {
			logger.Debug("no Content-Type header")
			span.SetStatus(codes.Error, "missing content-type header")

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		requestBody := struct {
			URL string `json:"url"`
		}{}

		if contentType[0] != "application/json" {
			logger.Debug("unexpected content-type", "type", contentType)
			span.SetStatus(codes.Error, "unexpected content-type")

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			logger.Debug("error parsing json request body no url field")
			span.RecordError(err)
			span.SetStatus(codes.Error, "error parsing json request body no url field")

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if len(requestBody.URL) == 0 {
			logger.Debug("error parsing json request body empty url")
			span.RecordError(errMissingURLField)
			span.SetStatus(codes.Error, "error parsing json request body empty url")

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if _, err := url.ParseRequestURI(requestBody.URL); err != nil {
			logger.Debug("error request body contains invalid url")
			span.SetStatus(codes.Error, "invalid url")
			span.RecordError(err)

			w.WriteHeader(http.StatusBadRequest)

			return
		}

		short, err := generateHash(ctx, requestBody.URL, logger, tracer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if err := store.AddLink(ctx, short, requestBody.URL); err != nil {
			logger.Error("error adding row into db", "err", err)
			span.SetStatus(codes.Error, "error adding row")
			span.RecordError(err)
		}

		link := Link{
			Short:    path.Join(r.Host, short),
			Original: requestBody.URL,
		}

		err = WriteJSON(w, http.StatusCreated, link)
		if err != nil {
			logger.Error("error writing JSON response", "err", err)
			span.SetStatus(codes.Error, "error writing JSON")
			span.RecordError(err)

			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func generateHash(ctx context.Context, url string, logger *slog.Logger, tracer trace.Tracer) (string, error) {
	_, span := tracer.Start(ctx, "generating_hash")
	defer span.End()

	logger = logger.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("url", url),
	)

	//nolint: gosec // md5 is fine for link shortening but probably slower than it could be
	h := md5.New()

	_, err := io.WriteString(h, url)
	if err != nil {
		logger.Error("error writing to hash", "err", err)
		span.SetStatus(codes.Error, "error writing to hash")
		span.RecordError(err)

		return "", fmt.Errorf("error writing hash: %w", err)
	}

	hash := h.Sum(nil)
	reader := bytes.NewReader(hash[:4])

	var x uint32
	if err := binary.Read(reader, binary.LittleEndian, &x); err != nil {
		logger.Error("error reading bytes", "err", err)
		span.RecordError(err)

		return "", fmt.Errorf("error reading bytes: %w", err)
	}

	short := base62.Encode(uint64(x))

	logger.Debug("hash generated",
		slog.String("hash", short),
	)

	return short, nil
}

func HandlerGetLink(logger *slog.Logger, store Store) http.HandlerFunc {
	tracer := otel.Tracer("handlercreatelink")

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "get_link")
		defer span.End()

		logger = logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
		)

		original, err := store.GetOriginal(ctx, r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))
			span.RecordError(err)
			span.SetStatus(codes.Error, "unknown link")

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
			span.RecordError(err)
			span.SetStatus(codes.Error, "error writing JSON response")

			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func HandlerRedirect(logger *slog.Logger, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		original, err := store.GetOriginal(r.Context(), r.PathValue("short"))
		if err != nil {
			logger.Info("unknown link", "short", r.PathValue("short"))

			w.WriteHeader(http.StatusNotFound)

			return
		}

		http.Redirect(w, r, *original, http.StatusTemporaryRedirect)
	}
}
