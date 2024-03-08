package links

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
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

func (mps *mockStore) addLink(short, url string) error {
	_, ok := mps.m[short]
	if ok {
		return fmt.Errorf("Short %s already exists", short)
	} else {
		mps.m[short] = mockRow{short, url}
		return nil
	}
}

func (mps *mockStore) getOriginal(short string) (*string, error) {
	row, ok := mps.m[short]
	if !ok {
		return nil, fmt.Errorf("Short %s does not exists", short)
	}
	return &row.original, nil
}

func Test_handlerAddLink(t *testing.T) {
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := handlerCreateLink(logger, store)

	t.Run("Add link to http://example.com", func(t *testing.T) {

		r := httptest.NewRequest("POST", "http://goshort.test/api/v1/links", strings.NewReader(`{"url":"http://example.com"}`))
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 200 {
			t.Errorf("expected StatusCode 200 got %d", w.Result().StatusCode)
		}
		defer w.Result().Body.Close()
		responseStruct := Link{}
		json.NewDecoder(w.Result().Body).Decode(&responseStruct)
		if responseStruct.Original != "http://example.com" {
			t.Errorf(`expected value "http://example.com" not stored got %s`, responseStruct.Original)
		}
		_, short := path.Split(responseStruct.Short)
		storedValue, err := store.getOriginal(short)
		if err != nil {
			t.Fatalf("cannot retrieve value %v", store.m)
		}
		if *storedValue != "http://example.com" {
			t.Errorf("data stored does not match got %s expected %s", *storedValue, "http://example.com")
		}
	})
	t.Run("Bad request no Content-Type", func(t *testing.T) {

		r := httptest.NewRequest("POST", "/api/v1/links", nil)
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 400 {
			t.Errorf("expected StatusCode 400 got %d", w.Result().StatusCode)
		}
	})
	t.Run("Bad request unexpected Content-Type", func(t *testing.T) {

		r := httptest.NewRequest("POST", "/api/v1/links", nil)
		r.Header.Add("Content-Type", "application/binary")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 400 {
			t.Errorf("expected StatusCode 400 got %d", w.Result().StatusCode)
		}
	})
	t.Run("Bad request nil body", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/api/v1/links", nil)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 400 {
			t.Errorf("expected StatusCode 400 got %d", w.Result().StatusCode)
		}
	})
	t.Run("Bad request empty url in json", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/api/v1/links", strings.NewReader(`{"url":""}`))
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 400 {
			t.Errorf("expected StatusCode 400 got %d", w.Result().StatusCode)
		}
	})
}

func Test_handlerGetLink(t *testing.T) {
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := handlerGetLink(logger, store)
	store.addLink("test", "http://example.com")
	t.Run(`Get "test" link to http://example.com`, func(t *testing.T) {

		r := httptest.NewRequest("GET", "/api/v1/links/test", nil)
		r.SetPathValue("short", "test")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 200 {
			t.Errorf("expected StatusCode 200 got %d", w.Result().StatusCode)
		}
		defer w.Result().Body.Close()
		responseStruct := Link{}
		json.NewDecoder(w.Result().Body).Decode(&responseStruct)
		if responseStruct.Original != "http://example.com" {
			t.Errorf(`expected value "http://example.com" not stored got %s`, responseStruct.Original)
		}
	})
	t.Run(`Get nonexistent link`, func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/links/test2", nil)
		r.SetPathValue("short", "test2")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 404 {
			t.Errorf("expected StatusCode 404 got %d", w.Result().StatusCode)
		}
	})
}

func Test_handlerRedirectLink(t *testing.T) {
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handlerFunc := handlerRedirect(logger, store)
	store.addLink("test", "http://example.com")
	t.Run(`Get "test" link to example.com`, func(t *testing.T) {

		r := httptest.NewRequest("GET", "/test", nil)
		r.SetPathValue("short", "test")
		w := httptest.NewRecorder()

		handlerFunc(w, r)

		if w.Result().StatusCode != 303 {
			t.Errorf("expected StatusCode 303 got %d", w.Result().StatusCode)
		}
		redirectLocation := w.Result().Header.Get("Location")
		if redirectLocation != "http://example.com" {
			t.Errorf("Redirect to unexpected location expected %s got %s", "http://example.com", redirectLocation)
		}
	})
	t.Run(`Get nonexistent link`, func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test2", nil)
		r.SetPathValue("short", "test2")
		w := httptest.NewRecorder()
		handlerFunc(w, r)
		if w.Result().StatusCode != 404 {
			t.Errorf("expected StatusCode 404 got %d", w.Result().StatusCode)
		}
	})
}
