# Language-model API keys entered in the app are sealed in Postgres, not written to Vault

An admin can add API keys for Anthropic, OpenRouter and Mistral in the app, keep more than one,
and choose which key and model product matching uses, without touching Vault. The running binary
has to read a key it was given at runtime. Writing it to Vault would keep every secret in one
place, but the pod's Vault identity would then need write access to its own path, Vault would gain
a second writer besides the operators, and a key would reach the process only after an External
Secrets sync and a restart. As in the tribelt mirror (its ADR-0005), the binary seals each key and
stores it in its own database, which the platform's backups already cover.

The key is sealed with AES-256-GCM under a key derived with HKDF-SHA256 from `SESSION_KEY` (info
`findability-explorer-llm-v1`), which already arrives from Vault for the sessions (ADR-0001), so no
new secret is needed. Each row records a key id derived from the same secret, and the row's id
and provider are bound into the ciphertext as associated data, so a sealed key cannot move to
another row. Keys are write-only: the API shows a fingerprint (the first 12 hex characters of the
key's SHA-256), the label, the provider, who added it and when, and its latest test, never the key.
A key is checked with its provider (by listing the models it may use) before it is stored; a
refused key is not stored. Every addition, deletion, test and change of the active model is
appended to `llm_events`.

The Vault-delivered `ANTHROPIC_API_KEY` keeps working as the read-only credential
`vault-anthropic`. Until an admin chooses, matching uses it with `claude-haiku-5-5`, as before; an
admin may choose another credential and model, or turn matching off. Deleting the active stored key
falls back to that default. Each analysis asks for the active credential and model when it runs,
and records them (`matchingModel`), so a change applies from the next analysis on.

## Consequences

- Rotating `SESSION_KEY` makes every stored key unreadable: the key id no longer matches, the
  credential shows `readable: false`, and nothing is decrypted with the wrong key. The admin
  deletes it and adds it again. A rotation without re-entry needs a one-off re-encryption (open
  with the old key, seal with the new) first; the key id finds the rows to convert.
- A database dump holds the sealed keys but not the key that opens them. Anyone holding both
  `SESSION_KEY` and a dump does.
- `SESSION_KEY` now guards the session cookie, the refresh tokens and the API keys; each use
  derives its own key with its own HKDF info string.
- The monthly token budget (`LLM_MONTHLY_TOKENS`) counts every provider's tokens alike.
- The process needs egress to `openrouter.ai` and `api.mistral.ai` besides `api.anthropic.com`.
