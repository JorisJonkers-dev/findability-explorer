package persistence_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

func TestAddThenList(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repo := persistence.New(store.Queries())
	u, _ := url.Parse("https://www.example-belts.com/")
	at := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	c := domain.Company{ID: "example-belts", Name: "Example Belts", URL: u, Host: "example-belts.com", AddedBy: "joris", AddedAt: at}
	if err := repo.Add(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if err := repo.Add(t.Context(), c); err == nil {
		t.Fatal("the same company was added twice")
	}
	clash := c
	clash.ID = "another-id"
	if err := repo.Add(t.Context(), clash); err == nil {
		t.Fatal("a second company on the same host was added")
	}
	got, err := repo.List(t.Context())
	if err != nil || len(got) != 1 {
		t.Fatalf("List = %+v, %v", got, err)
	}
	g := got[0]
	if g.ID != c.ID || g.Name != c.Name || g.URL.String() != u.String() || g.Website != u.String() || g.Host != c.Host ||
		g.Source != domain.SourceAdded || g.AddedBy != "joris" || !g.AddedAt.Equal(at) {
		t.Fatalf("listed %+v", g)
	}
}
