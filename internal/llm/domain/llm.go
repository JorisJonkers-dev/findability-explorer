// Package domain is the language-model context's pure core: the API keys (credentials) an admin
// added or Vault delivers, which credential and model product matching uses, the log of changes,
// and the port to the providers (docs/adr/0002). It imports no transport, persistence or
// generated code (enforced by depguard).
package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Provider is a language-model provider.
type Provider string

const (
	Anthropic  Provider = "anthropic"
	OpenRouter Provider = "openrouter"
	Mistral    Provider = "mistral"
	// AnthropicAdmin is an Anthropic Admin key (sk-ant-admin…): it only reads the organisation's
	// usage and cost reports, and never asks a model.
	AnthropicAdmin Provider = "anthropic_admin"
)

const (
	// VaultID is the credential Vault delivers in ANTHROPIC_API_KEY. It is read-only here.
	VaultID = "vault-anthropic"
	// DefaultModel is the model the Vault credential uses until an admin chooses: the current
	// Haiku, cheap enough for every analysis.
	DefaultModel = "claude-haiku-5-5"
	// MaxEvents is how many events the settings show.
	MaxEvents = 50

	maxLabel   = 100
	minKey     = 8
	maxKey     = 500
	maxModel   = 200
	maxMessage = 500
)

var (
	// ErrInvalid is returned for a credential or model that breaks a rule below.
	ErrInvalid = errors.New("invalid")
	// ErrInvalidProvider is returned for a provider other than anthropic, openrouter, mistral or
	// anthropic_admin.
	ErrInvalidProvider = fmt.Errorf("%w: the provider is anthropic, openrouter, mistral or anthropic_admin", ErrInvalid)
	// ErrReportsOnly is returned when an Admin key is chosen to answer: it only reads reports.
	ErrReportsOnly = fmt.Errorf("%w: an Anthropic Admin key reads usage reports and cannot match products", ErrInvalid)
	// ErrInvalidLabel is returned for a label that is blank or too long.
	ErrInvalidLabel = fmt.Errorf("%w: a label has 1 to %d characters", ErrInvalid, maxLabel)
	// ErrInvalidKey is returned for a key of the wrong length or with spaces or control characters.
	ErrInvalidKey = fmt.Errorf("%w: an API key has %d to %d characters and no spaces", ErrInvalid, minKey, maxKey)
	// ErrInvalidModel is returned for a model id that is blank, too long or has spaces.
	ErrInvalidModel = fmt.Errorf("%w: a model id has 1 to %d characters and no spaces", ErrInvalid, maxModel)
	// ErrNotFound is returned for a credential id nobody added.
	ErrNotFound = errors.New("no credential has this id")
	// ErrUnreadable is returned for a stored key sealed under another SESSION_KEY.
	ErrUnreadable = errors.New("the key was sealed under another SESSION_KEY; delete it and add it again")
	// ErrVaultReadOnly is returned for a change to the Vault credential.
	ErrVaultReadOnly = errors.New("the Vault key is managed in Vault and cannot be deleted here")
	// ErrKeyRefused is returned when the provider refuses a key being added.
	ErrKeyRefused = errors.New("the provider refused the key")
	// ErrProvider is returned when the provider fails or refuses a request.
	ErrProvider = errors.New("the provider failed")
)

// ParseProvider returns the provider named s.
func ParseProvider(s string) (Provider, error) {
	switch p := Provider(s); p {
	case Anthropic, OpenRouter, Mistral, AnthropicAdmin:
		return p, nil
	default:
		return "", ErrInvalidProvider
	}
}

// Source is where a credential comes from.
type Source string

const (
	SourceStored Source = "stored"
	SourceVault  Source = "vault"
)

// Test is the outcome of asking a provider whether it accepts a key.
type Test struct {
	OK      bool
	At      time.Time
	Message string
}

// Credential is an API key, without its secret.
type Credential struct {
	ID          string
	Provider    Provider
	Label       string
	Source      Source
	Fingerprint string
	// Readable is false for a stored key sealed under another SESSION_KEY.
	Readable  bool
	CreatedBy string    // empty for the Vault key
	CreatedAt time.Time // zero for the Vault key
	LastTest  *Test
}

// VaultCredential is the credential of the key Vault delivers.
func VaultCredential(key string) Credential {
	return Credential{
		ID: VaultID, Provider: Anthropic, Label: "Managed by Vault (ANTHROPIC_API_KEY)", Source: SourceVault,
		Fingerprint: Fingerprint(key), Readable: true,
	}
}

// Fingerprint is the first 12 hex characters of key's SHA-256: enough to tell keys apart, never
// enough to recover one.
func Fingerprint(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12]
}

// NewCredential is a key an admin is adding, checked.
type NewCredential struct {
	Provider Provider
	Label    string
	Key      string
}

// CheckNew validates a key an admin is adding; it trims the label and the key.
func CheckNew(provider, label, key string) (NewCredential, error) {
	p, err := ParseProvider(provider)
	if err != nil {
		return NewCredential{}, err
	}
	label = strings.TrimSpace(label)
	if n := len([]rune(label)); n < 1 || n > maxLabel || strings.ContainsFunc(label, unicode.IsControl) {
		return NewCredential{}, ErrInvalidLabel
	}
	key = strings.TrimSpace(key)
	if n := len(key); n < minKey || n > maxKey || strings.ContainsFunc(key, notKeyRune) {
		return NewCredential{}, ErrInvalidKey
	}
	return NewCredential{Provider: p, Label: label, Key: key}, nil
}

func notKeyRune(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }

// CheckModel validates a model id.
func CheckModel(model string) (string, error) {
	if n := len(model); n < 1 || n > maxModel || strings.ContainsFunc(model, notKeyRune) {
		return "", ErrInvalidModel
	}
	return model, nil
}

// Active is the credential and model product matching uses.
type Active struct {
	CredentialID string
	Model        string
}

// Mode is what an admin decided about the active model.
type Mode string

const (
	// ModeUnset means nobody decided, or the chosen credential was deleted.
	ModeUnset Mode = ""
	// ModeChosen means an admin chose a credential and model.
	ModeChosen Mode = "chosen"
	// ModeOff means an admin turned matching off.
	ModeOff Mode = "off"
)

// Setting is the stored decision about the active model.
type Setting struct {
	Mode   Mode
	Active Active // only with ModeChosen
}

// Choice is how the active model was decided, as the settings show it.
type Choice string

const (
	ChoiceDefault Choice = "default"
	ChoiceChosen  Choice = "chosen"
	ChoiceOff     Choice = "off"
)

// Resolve is the credential and model to use under s, nil when matching is off. Until an admin
// decides, that is the Vault key with DefaultModel when Vault delivers one.
func Resolve(s Setting, vault bool) (Choice, *Active) {
	switch s.Mode {
	case ModeChosen:
		a := s.Active
		return ChoiceChosen, &a
	case ModeOff:
		return ChoiceOff, nil
	case ModeUnset:
	}
	if vault {
		return ChoiceDefault, &Active{CredentialID: VaultID, Model: DefaultModel}
	}
	return ChoiceDefault, nil
}

// Describe names the provider and model, as an analysis records it.
func Describe(p Provider, model string) string { return string(p) + ": " + model }

// Action is what an event records.
type Action string

//nolint:gosec // action names, not credentials
const (
	ActionCredentialAdded   Action = "credential_added"
	ActionCredentialDeleted Action = "credential_deleted"
	ActionCredentialTested  Action = "credential_tested"
	ActionActiveSet         Action = "active_set"
	ActionActiveCleared     Action = "active_cleared"
)

// Event is one change to or test of the settings.
type Event struct {
	At           time.Time
	Actor        string
	Action       Action
	CredentialID string   // empty when none
	Provider     Provider // empty when none
	Model        string   // empty when none
	OK           *bool    // only for a test or an added key
	Message      string
}

// Clip shortens a message to what an event or a test stores.
func Clip(message string) string {
	if r := []rune(message); len(r) > maxMessage {
		return string(r[:maxMessage-1]) + "…"
	}
	return message
}

// Model is a model a key may use.
type Model struct {
	ID   string
	Name string
}

// Prompt is one request for an answer that follows a JSON schema.
type Prompt struct {
	Model      string
	System     string
	User       string
	SchemaName string
	Schema     map[string]any
	MaxTokens  int64
}

// Completion is a provider's answer and what it consumed.
type Completion struct {
	Text         string
	InputTokens  int64
	OutputTokens int64
}

// Client is the port a provider adapter implements, for one key.
type Client interface {
	// Models lists the models the key may use; it fails when the provider refuses the key.
	Models(ctx context.Context) ([]Model, error)
	// Complete asks model for an answer. On an error the Completion still counts what was used.
	Complete(ctx context.Context, p Prompt) (Completion, error)
}

// Connector makes a Client for a provider and key.
type Connector interface {
	Connect(p Provider, key string) Client
}

// AnthropicKey is the Anthropic key cheap background work uses: the first readable one, in the
// order given (the Vault key first), whose latest test did not fail.
func AnthropicKey(creds []Credential) (Credential, bool) {
	for _, c := range creds {
		if c.Provider == Anthropic && c.Readable && (c.LastTest == nil || c.LastTest.OK) {
			return c, true
		}
	}
	return Credential{}, false
}
