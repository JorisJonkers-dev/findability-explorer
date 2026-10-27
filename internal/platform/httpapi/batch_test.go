package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// batchPorts records nothing and scans every company it is handed.
type batchPorts struct{}

func (batchPorts) RecordAdded(context.Context, string, string, string) error { return nil }

func (batchPorts) Scan(_ context.Context, _ string, ids []string) (int, error) { return len(ids), nil }

func TestAddingSeveralCompetitorsIsForAdministrators(t *testing.T) {
	srv := serve(t, &analyses{})
	body := `{"items":[{"name":"Intralox","url":"https://www.intralox.com","candidateId":"3a7bd3e2360a9f01"},` +
		`{"name":"Twente","url":"https://www.twentebelt.com"},{"name":"Example Belts"}],"scan":true}`
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/competitors/batch", body, "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer added companies: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/competitors/batch", body, "admin")
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 3 || r.body["scans"].(float64) != 1 {
		t.Fatalf("batch = %d %v", r.code, r.body)
	}
	added, dup, rejected := items[0].(map[string]any), items[1].(map[string]any), items[2].(map[string]any)
	if added["outcome"] != "added" || added["competitorId"] != "intralox" || dup["outcome"] != "duplicate" || dup["competitorId"] != "twentebelt" ||
		rejected["outcome"] != "rejected" || rejected["reason"] != "Add its website first." || rejected["competitorId"] != nil {
		t.Errorf("items = %v", items)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/competitors/batch", `{"items":[]}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("an empty batch = %d", r.code)
	}
}
