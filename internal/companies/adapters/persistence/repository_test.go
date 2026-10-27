package persistence_test

import (
	"errors"
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

func TestUpdateDeleteAndOverrides(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store.Queries())
	u, _ := url.Parse("https://www.changing.example/")
	at := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	c := domain.Company{ID: "changing", Name: "Changing", URL: u, Host: "changing.example", AddedBy: "joris", AddedAt: at}
	if err := repo.Add(ctx, c); err != nil {
		t.Fatal(err)
	}
	moved, _ := url.Parse("https://moved.example/")
	c.Name, c.URL, c.Host = "Changed", moved, "moved.example"
	if err := repo.Update(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := repo.List(ctx)
	if err != nil || len(got) != 1 || got[0].Name != "Changed" || got[0].Host != "moved.example" {
		t.Fatalf("after Update = %+v, %v", got, err)
	}
	// An added company is archived in its row, and a person keeps it when restoring it.
	c.Archived, c.ArchivedBy, c.ArchivedAt = true, "language model", at
	if err := repo.Update(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err = repo.List(ctx)
	if err != nil || !got[0].Archived || got[0].ArchivedBy != "language model" || !got[0].ArchivedAt.Equal(at) || got[0].KeptBy != "" {
		t.Fatalf("after archiving = %+v, %v", got, err)
	}
	c.Archived, c.ArchivedBy, c.KeptBy, c.KeptAt = false, "language model", "joris", at.Add(time.Hour)
	if err := repo.Update(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err = repo.List(ctx)
	if err != nil || got[0].Archived || got[0].ArchivedBy != "" || !got[0].ArchivedAt.IsZero() || got[0].KeptBy != "joris" || !got[0].KeptAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("after restoring = %+v, %v", got, err)
	}
	if err := repo.Update(ctx, domain.Company{ID: "nobody", URL: moved}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update(nobody) = %v", err)
	}
	if err := repo.Delete(ctx, "changing"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "changing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second Delete = %v", err)
	}

	site, _ := url.Parse("https://www.twentebelt.nl/")
	o := domain.Override{ID: "twentebelt", Name: "Twentebelt B.V.", Website: site, Host: "twentebelt.nl", ArchivedBy: "joris", ArchivedAt: at, UpdatedBy: "joris", UpdatedAt: at}
	if err := repo.SaveOverride(ctx, o); err != nil {
		t.Fatal(err)
	}
	restored := domain.Override{ID: "twentebelt", KeptBy: "joris", KeptAt: at.Add(time.Hour), UpdatedBy: "joris", UpdatedAt: at.Add(time.Hour)}
	if err := repo.SaveOverride(ctx, domain.Override{ID: "martens", Name: "Märtens", UpdatedBy: "j", UpdatedAt: at}); err != nil {
		t.Fatal(err)
	}
	overrides, err := repo.Overrides(ctx)
	if err != nil || len(overrides) != 2 || overrides[1].ID != "twentebelt" || overrides[1].Website.String() != site.String() ||
		!overrides[1].ArchivedAt.Equal(at) || overrides[0].Website != nil || !overrides[0].ArchivedAt.IsZero() {
		t.Fatalf("Overrides = %+v, %v", overrides, err)
	}
	if err := repo.SaveOverride(ctx, restored); err != nil {
		t.Fatal(err)
	}
	overrides, err = repo.Overrides(ctx)
	if err != nil || overrides[1].Name != "" || !overrides[1].ArchivedAt.IsZero() || !overrides[1].UpdatedAt.Equal(at.Add(time.Hour)) ||
		overrides[1].KeptBy != "joris" || !overrides[1].KeptAt.Equal(at.Add(time.Hour)) || overrides[0].KeptBy != "" {
		t.Fatalf("after restore = %+v, %v", overrides, err)
	}
}
