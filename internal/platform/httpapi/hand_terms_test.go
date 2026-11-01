package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// handStore is an in-memory store of the terms added and removed by hand; fail fails each call.
type handStore struct {
	added   []terminologydomain.HandTerm
	removed []terminologydomain.Removal
	fail    bool
}

func (a *analyses) handStore() *handStore {
	if a.hand == nil {
		a.hand = &handStore{}
	}
	return a.hand
}

var errHand = errors.New("store down")

func (h *handStore) err() error {
	if h.fail {
		return errHand
	}
	return nil
}

func (h *handStore) Added(context.Context) ([]terminologydomain.HandTerm, error) {
	return h.added, h.err()
}

func (h *handStore) Add(_ context.Context, t terminologydomain.HandTerm) error {
	h.added = append(h.added, t)
	return h.err()
}

func (h *handStore) Removed(context.Context) ([]terminologydomain.Removal, error) {
	return h.removed, h.err()
}

func (h *handStore) Remove(_ context.Context, r terminologydomain.Removal) error {
	h.removed = append(h.removed, r)
	return h.err()
}

func (h *handStore) PutBack(_ context.Context, key terminologydomain.TermKey) (bool, error) {
	n := len(h.removed)
	h.removed = slices.DeleteFunc(h.removed, func(r terminologydomain.Removal) bool { return r.Key == key })
	return len(h.removed) < n, h.err()
}

func TestAddRemoveAndPutBackATermByHand(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	body := `{"term":"oven conveyor belt","language":"en","family":"eyelink","tier":"application"}`
	if r := call(t, srv, http.MethodPost, "/api/v1/admin/terms/hand", body, true); r.code != http.StatusForbidden {
		t.Fatalf("a reader adding a term: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/terms/hand", body, "admin")
	if r.code != http.StatusCreated || r.body["term"] != "oven conveyor belt" || r.body["origin"] != "hand" {
		t.Fatalf("add = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/terms/hand", body, "admin"); r.code != http.StatusConflict {
		t.Errorf("adding twice: %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/terms/hand", `{"term":"x","language":"fr","family":"other","tier":"broad"}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("a French term: %d", r.code)
	}
	// Every term marks it as added by hand.
	search := call(t, srv, http.MethodGet, "/api/v1/terms?q=oven%20conveyor%20belt", "", true)
	if items, _ := search.body["items"].([]any); len(items) != 1 || items[0].(map[string]any)["origin"] != "hand" {
		t.Errorf("search = %v", search.body)
	}

	ref := `{"term":"OVEN conveyor belt","language":"en"}`
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/terms/removed", ref, "admin"); r.code != http.StatusNoContent {
		t.Fatalf("remove = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/terms/removed", ref, "admin"); r.code != http.StatusNotFound {
		t.Errorf("removing twice: %d", r.code)
	}
	list := callAs(t, srv, http.MethodGet, "/api/v1/admin/terms/hand", "", "admin")
	added, _ := list.body["added"].([]any)
	removed, _ := list.body["removed"].([]any)
	if list.code != http.StatusOK || len(added) != 1 || len(removed) != 1 || removed[0].(map[string]any)["by"] == "" {
		t.Fatalf("list = %d %v", list.code, list.body)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/admin/terms/hand", "", true); r.code != http.StatusForbidden {
		t.Errorf("a reader listing: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodDelete, "/api/v1/admin/terms/removed?term=oven%20conveyor%20belt&language=en", "", "admin"); r.code != http.StatusNoContent {
		t.Fatalf("put back = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodDelete, "/api/v1/admin/terms/removed?term=oven%20conveyor%20belt&language=en", "", "admin"); r.code != http.StatusNotFound {
		t.Errorf("putting back twice: %d", r.code)
	}

	a.hand.fail = true
	for _, c := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/terms/hand", `{"term":"belt two","language":"en","family":"other","tier":"broad"}`},
		{http.MethodPost, "/api/v1/admin/terms/removed", `{"term":"oven conveyor belt","language":"en"}`},
		{http.MethodDelete, "/api/v1/admin/terms/removed?term=x&language=en", ""},
	} {
		if r := callAs(t, srv, c.method, c.path, c.body, "admin"); r.code != http.StatusInternalServerError {
			t.Errorf("%s %s with a failing store: %d", c.method, c.path, r.code)
		}
	}
}
