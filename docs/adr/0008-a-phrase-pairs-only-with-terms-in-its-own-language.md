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

- Stored results are not rewritten; they are read by the rule instead.
  - One rule says when a phrase is a term's spelling (`internal/shared/spelling`), and both the
    scans and the review queue use it.
  - The review queue reads every keyword of the finished scans, not only those stored outside the
    map. It decides itself whether the map holds each phrase in its language, so a keyword stored
    as in the map through another language's term comes back as a candidate. It also drops a
    closest term the map holds only in another language. The tier and family it suggests stay as
    the scan stored them; classifying every finding again on each read of the queue costs too much.
  - A scan's page, its company's profile and the comparison of two scans classify each stored
    keyword again from its phrase and language, by the map as it is now. Tier, family, closest term
    and "in the map" so follow the rule, and also any terms the map gained since.
  - Known terms are stored without the page they were counted on, only with the scan's pages and
    their languages. A known term, and the company it credits as using the term, is left out when
    the scan read no page in the term's language. One counted on a German page and also on an
    English one cannot be told apart from one counted on the German page alone, so it stays.
- Decisions and verdicts made before carry the phrase's old id. They still count where they apply:
  a rejection for the phrase in every language, an acceptance only in the language of the term it
  made. A candidate an old acceptance does not apply to waits for review again, and the language
  model is asked about it once more.
