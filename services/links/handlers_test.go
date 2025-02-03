package links_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jacekdobrowolski/goshort/services/links"
)

type mockRow struct {
	short    string
	original string
}

type mockStore struct {
	m map[string]mockRow
}

func newMockStore() *mockStore {
	return &mockStore{
		m: make(map[string]mockRow),
	}
}

var errShortExists = errors.New("short already exists")

func (mps *mockStore) AddLink(short, url string) error {
	_, ok := mps.m[short]
	if ok {
		return fmt.Errorf("%w %s", errShortExists, short)
	}

	mps.m[short] = mockRow{
		short:    short,
		original: url,
	}

	return nil
}

var errShortDoesntExist = errors.New("short does not exists")

func (mps *mockStore) GetOriginal(short string) (*string, error) {
	row, ok := mps.m[short]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errShortDoesntExist, short)
	}

	return &row.original, nil
}

func Test_handlerAddLink(t *testing.T) {
	t.Parallel()

	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := links.HandlerCreateLink(logger, store)

	t.Run("Add link to http://example.com", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost,
			"http://goshort.test/api/v1/links",
			strings.NewReader(`{"url":"http://example.com"}`),
		)
		r.Header.Add("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusCreated {
			t.Errorf("expected StatusCode %d got %d", http.StatusCreated, w.Result().StatusCode)
		}

		defer w.Result().Body.Close()

		responseStruct := links.Link{}

		err := json.NewDecoder(w.Result().Body).Decode(&responseStruct)
		if err != nil {
			t.Error(err)
		}

		if responseStruct.Original != "http://example.com" {
			t.Errorf(`expected value "http://example.com" not stored got %s`, responseStruct.Original)
		}

		_, short := path.Split(responseStruct.Short)

		storedValue, err := store.GetOriginal(short)
		if err != nil {
			t.Fatalf("cannot retrieve value %v", store.m)
		}

		if *storedValue != "http://example.com" {
			t.Errorf("data stored does not match got %s expected %s", *storedValue, "http://example.com")
		}
	})

	t.Run("Bad request no Content-Type", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected StatusCode %d got %d", http.StatusBadRequest, w.Result().StatusCode)
		}
	})

	t.Run("Bad request unexpected Content-Type", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/v1/links", nil)
		r.Header.Add("Content-Type", "application/binary")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected StatusCode %d got %d", http.StatusBadRequest, w.Result().StatusCode)
		}
	})

	t.Run("Bad request nil body", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/v1/links", nil)
		r.Header.Add("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected StatusCode %d got %d", http.StatusBadRequest, w.Result().StatusCode)
		}
	})

	t.Run("Bad request empty url in json", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":""}`))
		r.Header.Add("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected StatusCode %d got %d", http.StatusBadRequest, w.Result().StatusCode)
		}
	})
}

func Fuzz_handlerAddLink(f *testing.F) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelError,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			return a
		},
	}))

	validBase62RegEx := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	f.Add(`{"url":"http://example.com"}`, "Content-Type", "application/json")

	f.Fuzz(func(t *testing.T, body string, headerKey string, headerValue string) {
		store := newMockStore()

		handlerFunc := links.HandlerCreateLink(logger, store)

		r := httptest.NewRequest(http.MethodPost, "http://goshort.test/api/v1/links", strings.NewReader(body))
		r.Header.Add(headerKey, headerValue)

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusCreated &&
			w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected StatusCode %d or %d got %d", http.StatusCreated, http.StatusBadRequest, w.Result().StatusCode)
		}

		if w.Result().StatusCode != http.StatusCreated {
			return
		}

		defer w.Result().Body.Close()

		responseStruct := links.Link{}

		err := json.NewDecoder(w.Result().Body).Decode(&responseStruct)
		if err != nil {
			t.Fatal("error decoding response struct:", err)
		}

		if !utf8.ValidString(responseStruct.Original) {
			t.Errorf(`parsed value is not valid UTF-8 string %s`, responseStruct.Original)
		}

		_, short := path.Split(responseStruct.Short)
		if !validBase62RegEx.MatchString(short) {
			t.Errorf(`returned short value contains non alphanumeric characters %s`, short)
		}

		storedValue, err := store.GetOriginal(short)
		if err != nil {
			t.Fatalf("cannot retrieve value %v", store.m)
		}

		if _, err := url.ParseRequestURI(*storedValue); err != nil {
			t.Fatalf("stored data is not a valid URL: %s", *storedValue)
		}
	})
}

func Test_handlerGetLink(t *testing.T) {
	t.Parallel()

	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := links.HandlerGetLink(logger, store)

	err := store.AddLink("test", "http://example.com")
	if err != nil {
		t.Fatal("error adding link", err)
	}

	t.Run(`Get "test" link to http://example.com`, func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/api/v1/links/test", nil)
		r.SetPathValue("short", "test")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected StatusCode %d got %d", http.StatusOK, w.Result().StatusCode)
		}

		defer w.Result().Body.Close()

		responseStruct := links.Link{}

		err := json.NewDecoder(w.Result().Body).Decode(&responseStruct)
		if err != nil {
			t.Fatal("error decoding reposnse struct:", err)
		}

		if responseStruct.Original != "http://example.com" {
			t.Errorf(`expected value "http://example.com" not stored got %s`, responseStruct.Original)
		}
	})
	t.Run(`Get nonexistent link`, func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/api/v1/links/test2", nil)
		r.SetPathValue("short", "test2")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected StatusCode %d got %d", http.StatusNotFound, w.Result().StatusCode)
		}
	})
}

func Test_handlerRedirectLink(t *testing.T) {
	t.Parallel()

	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := links.HandlerRedirect(logger, store)

	err := store.AddLink("test", "http://example.com")
	if err != nil {
		t.Fatal("error adding link", err)
	}

	t.Run(`Get "test" link to example.com`, func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.SetPathValue("short", "test")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusTemporaryRedirect {
			t.Errorf("expected StatusCode %d got %d", http.StatusTemporaryRedirect, w.Result().StatusCode)
		}

		redirectLocation := w.Result().Header.Get("Location")
		if redirectLocation != "http://example.com" {
			t.Errorf("Redirect to unexpected location expected %s got %s", "http://example.com", redirectLocation)
		}
	})

	t.Run(`Get nonexistent link`, func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/test2", nil)
		r.SetPathValue("short", "test2")

		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected StatusCode %d got %d", http.StatusNotFound, w.Result().StatusCode)
		}
	})
}
