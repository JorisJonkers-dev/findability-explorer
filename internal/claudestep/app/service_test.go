package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

var at = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

type repo struct {
	versions map[domain.ID][]domain.Version
	readErr  error
	saveErr  error
}

func (r *repo) Versions(_ context.Context, step domain.ID) ([]domain.Version, error) {
	return r.versions[step], r.readErr
}

func (r *repo) Save(_ context.Context, v domain.Version, original *domain.Version) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	if original != nil {
		r.versions[v.Step] = append(r.versions[v.Step], *original)
	}
	r.versions[v.Step] = append(r.versions[v.Step], v)
	return nil
}

func newService(r *repo, base llm.Micros, baseErr error) *app.Service {
	return app.New(r, func(context.Context, domain.ID) (llm.Micros, error) { return base, baseErr }, func() time.Time { return at })
}

func TestAStepWithoutSavedVersionsUsesTheInstructionsItCameWith(t *testing.T) {
	svc := newService(&repo{versions: map[domain.ID][]domain.Version{}}, 4_400, nil)
	st, err := svc.Step(t.Context(), domain.Matching)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Number != 1 || st.Current.Model != llm.DefaultModel || len(st.Versions) != 1 || st.JobCost != 4_400 {
		t.Errorf("step = %+v", st)
	}
	if st.Models[0].Info.ID != llm.DefaultModel || st.Models[0].JobCost != 4_400 || st.Models[len(st.Models)-1].JobCost <= 4_400 {
		t.Errorf("models = %+v", st.Models)
	}
	v, err := svc.Current(t.Context(), domain.Matching)
	if err != nil || v.Number != 1 || !strings.Contains(v.Instructions, "Tribelt") {
		t.Errorf("current = %+v %v", v, err)
	}
}

func TestASavedVersionIsUsedNextAndTheOriginalStaysInTheHistory(t *testing.T) {
	r := &repo{versions: map[domain.ID][]domain.Version{}}
	svc := newService(r, 4_400, nil)
	st, err := svc.Save(t.Context(), domain.Matching, "Joris", domain.Change{Model: "claude-sonnet-5-5", Instructions: "Match the metal belts."})
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Number != 2 || st.Current.By != "Joris" || len(st.Versions) != 2 || st.Versions[0].By != domain.ByTheTool || !st.Versions[0].At.Equal(at) {
		t.Errorf("after the first save = %+v", st)
	}
	if st.JobCost <= 4_400 {
		t.Errorf("Sonnet costs %d, no more than Haiku", st.JobCost)
	}
	v, err := svc.Current(t.Context(), domain.Matching)
	if err != nil || v.Model != "claude-sonnet-5-5" || v.Instructions != "Match the metal belts." {
		t.Errorf("the next job uses %+v %v", v, err)
	}
	// Restoring version 1 copies it into a new version; nothing is overwritten.
	st, err = svc.Save(t.Context(), domain.Matching, "Joris", domain.Change{Model: llm.DefaultModel, Instructions: st.Versions[0].Instructions, RestoredFrom: 1})
	if err != nil || st.Current.Number != 3 || st.Current.Note != "Restored from version 1." || len(r.versions[domain.Matching]) != 3 {
		t.Errorf("after the restore = %+v %v", st.Current, err)
	}
	steps, err := svc.Steps(t.Context())
	if err != nil || len(steps) != 5 || steps[1].Definition.ID != domain.Matching || steps[1].Current.Number != 3 || steps[0].Current.Number != 1 {
		t.Errorf("steps = %+v %v", steps, err)
	}
}

func TestAnUnknownStepOrAnInvalidVersionIsRefused(t *testing.T) {
	svc := newService(&repo{versions: map[domain.ID][]domain.Version{}}, 0, nil)
	if _, err := svc.Step(t.Context(), "nothing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("step: %v", err)
	}
	if _, err := svc.Current(t.Context(), "nothing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("current: %v", err)
	}
	if _, err := svc.Save(t.Context(), "nothing", "Joris", domain.Change{}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("save: %v", err)
	}
	if _, err := svc.Save(t.Context(), domain.Matching, "Joris", domain.Change{Model: "gpt-5", Instructions: "x"}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("invalid: %v", err)
	}
}

func TestStoreAndEstimateFailuresAreReported(t *testing.T) {
	broken := errors.New("database down")
	ch := domain.Change{Model: "claude-sonnet-5-5", Instructions: "x"}
	read := newService(&repo{versions: map[domain.ID][]domain.Version{}, readErr: broken}, 0, nil)
	if _, err := read.Steps(t.Context()); !errors.Is(err, broken) {
		t.Errorf("steps: %v", err)
	}
	if _, err := read.Current(t.Context(), domain.Matching); !errors.Is(err, broken) {
		t.Errorf("current: %v", err)
	}
	if _, err := read.Save(t.Context(), domain.Matching, "Joris", ch); !errors.Is(err, broken) {
		t.Errorf("save read: %v", err)
	}
	save := newService(&repo{versions: map[domain.ID][]domain.Version{}, saveErr: broken}, 0, nil)
	if _, err := save.Save(t.Context(), domain.Matching, "Joris", ch); !errors.Is(err, broken) {
		t.Errorf("save: %v", err)
	}
	base := newService(&repo{versions: map[domain.ID][]domain.Version{}}, 0, broken)
	if _, err := base.Step(t.Context(), domain.Matching); !errors.Is(err, broken) {
		t.Errorf("base: %v", err)
	}
}
