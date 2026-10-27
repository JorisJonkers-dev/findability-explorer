# A phrase pairs only with terms in its own language

The terminology map holds terms in Dutch, German and English. The scans compared a site's phrases
with every term of the map, whatever its language, and so did the review queue. A Dutch phrase
spelled like a German term ("transportband") counted as in the map, an English phrase could be
closest to a German term, a German term counted as used on an English page, and a phrase found in
Dutch and in German was one candidate, so accepting it as German hid its Dutch use.

**A phrase matches, is closest to, or is merged with a term only when both are in the same
language.** Two languages are the same when their primary subtags agree, so American English
(`en-US`) is English (`en`). A language that could not be told (`und`, or none) is the same as no
language, not even itself, so it matches nothing. `lang.Same` in `internal/shared/lang` is the one
place this rule lives.

- The lexicon (`internal/analysis/domain/lexicon.go`) finds a phrase's spelling, the spelling
  inside it and its closest term among the terms in the phrase's language only. A phrase in no
  language the map has is broad, with no closest term. How specific a word is, and which belt
  family it points at, are still counted over the whole map, since they describe words, not terms.
- `FindKnownTerms` counts a term only on the pages in its language. A page's language is its
  declared `<html lang>`, else the language whose stopwords it uses most; a page told by neither
  counts no term.
- The review queue keeps each language's use of a phrase as its own candidate, with an id made
  from the phrase and the language. The map holds a phrase only when it has the phrase in that
  language, a closest term in another language is dropped, and a candidate becomes a term only in
  its own language, whether a person or the language model accepts it.
- The web app never pairs a term with Tribelt's term in another language either: what Tribelt
  calls the same thing is its term in the same language, or it lacks it.

## Consequences

- Stored results are not rewritten. Scans made before this rule keep the `in_map` and
  `closest_term` their keywords were given, and the known terms they found, and the scan pages show
  them as stored. The review queue corrects what it can when it reads them: it drops a closest term
  the map holds only in another language. A keyword stored as in the map through another
  language's term stays out of the queue until the company is scanned again.
- Decisions and verdicts made before carry the phrase's old id. They still count where they apply:
  a rejection for the phrase in every language, an acceptance only in the language of the term it
  made. A candidate an old acceptance does not apply to waits for review again, and the language
  model is asked about it once more.
