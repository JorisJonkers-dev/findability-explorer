// Package seal encrypts secrets the service keeps (refresh tokens, language-model API keys) and
// the values it hands the browser in cookies, with AES-256-GCM under keys derived from SESSION_KEY
// (docs/adr/0002). Each use derives its own key with its own HKDF info string, so a blob sealed
// for one use never opens as another.
package seal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// MinSecretLength is the shortest SESSION_KEY the service accepts.
const MinSecretLength = 32

// Errors. None carries any detail of the sealed value.
var (
	// ErrShortSecret is returned for a SESSION_KEY shorter than MinSecretLength.
	ErrShortSecret = errors.New("SESSION_KEY must be at least 32 characters")
	// ErrKeyChanged is returned for a blob sealed under another key: SESSION_KEY was rotated.
	ErrKeyChanged = errors.New("sealed under another key")
	// ErrTampered is returned for a blob that does not open: altered, truncated, or bound to other data.
	ErrTampered = errors.New("does not open")
)

// Sealer encrypts secrets kept at rest. Every blob records the key id it was sealed under.
type Sealer struct {
	aead  cipher.AEAD
	keyID string
	rand  io.Reader
}

// NewSealer derives an AES-256 key from secret with HKDF-SHA256 and info, and a key id from the
// same secret with info + " key-id", so the id says nothing about the key itself.
func NewSealer(secret, info string) (*Sealer, error) {
	aead, err := newAEAD(secret, info)
	if err != nil {
		return nil, err
	}
	id, err := hkdf.Key(sha256.New, []byte(secret), nil, info+" key-id", 4)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead, keyID: "v1-" + hex.EncodeToString(id), rand: rand.Reader}, nil
}

func newAEAD(secret, info string) (cipher.AEAD, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrShortSecret
	}
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, info, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// KeyID names the key Seal uses now.
func (s *Sealer) KeyID() string { return s.keyID }

// Seal encrypts plain with bound as associated data: the blob opens only with the same bound.
func (s *Sealer) Seal(bound string, plain []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(s.rand, nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, plain, []byte(bound)), nil
}

// Open decrypts a blob sealed under keyID with bound.
func (s *Sealer) Open(bound, keyID string, blob []byte) ([]byte, error) {
	if keyID != s.keyID {
		return nil, ErrKeyChanged
	}
	return open(s.aead, bound, blob)
}

func open(aead cipher.AEAD, bound string, blob []byte) ([]byte, error) {
	n := aead.NonceSize()
	if len(blob) < n {
		return nil, ErrTampered
	}
	plain, err := aead.Open(nil, blob[:n], blob[n:], []byte(bound))
	if err != nil {
		return nil, ErrTampered
	}
	return plain, nil
}

// Codec seals values into cookie strings that expire; the cookie name is bound in, so a value
// sealed for one cookie never opens as another.
type Codec struct {
	aead cipher.AEAD
	rand io.Reader
}

// NewCodec derives the cookie key from secret with HKDF-SHA256 and info.
func NewCodec(secret, info string) (*Codec, error) {
	aead, err := newAEAD(secret, info)
	if err != nil {
		return nil, err
	}
	return &Codec{aead: aead, rand: rand.Reader}, nil
}

type envelope struct {
	Exp  int64           `json:"exp"`
	Data json.RawMessage `json:"d"`
}

// Seal encrypts v for the named cookie until now+ttl.
func (c *Codec) Seal(name string, v any, now time.Time, ttl time.Duration) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(envelope{Exp: now.Add(ttl).Unix(), Data: data})
	if err != nil {
		return "", err
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(c.aead.Seal(nonce, nonce, plain, []byte(name))), nil
}

// Open decrypts a value sealed for the named cookie into v, and refuses it once expired.
func (c *Codec) Open(name, sealed string, now time.Time, v any) error {
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		return ErrTampered
	}
	plain, err := open(c.aead, name, raw)
	if err != nil {
		return err
	}
	var env envelope
	if json.Unmarshal(plain, &env) != nil || now.Unix() >= env.Exp || json.Unmarshal(env.Data, v) != nil {
		return ErrTampered
	}
	return nil
}
