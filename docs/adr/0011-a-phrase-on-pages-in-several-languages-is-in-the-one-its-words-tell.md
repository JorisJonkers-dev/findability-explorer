# A phrase on pages in several languages is in the one its words tell

A scan gave each phrase the language of most of the pages it was found on. On a tie the
alphabetically first language won, so a phrase found on one Dutch and one German page was German,
and a page whose stopwords were as English as German was English. A phrase on pages whose language
could not be told was in no language (`und`), which ADR-0008 pairs with nothing, and the review
queue dropped it. On 22 competitor sites crawled in October 2026 (519 pages, 8,821 phrases), 95%
of the pages declared their language, so phrases in no language were rare (2). Ties were not: 90
phrases sat on as many pages in one language as in another, and most of them went to German by the
alphabet. "Metalen banden", "draadogenbanden" and "eye link belts" were German.

**A phrase found on pages in several languages, or in none, is in the language its own words
tell.** `lang.Vocabulary` in `internal/shared/lang` is the one place this rule lives. It reads
each word of a phrase:

- its letters: ä, ö, ü and ß are German, ij, ë and ï Dutch, and letters such as é, ñ, ç or ł
  another language;
- the word itself, in a short word list (`words.go`) or among the words of the terminology map's
  terms in each language, the list first;
- the known word it ends in, as compounds do ("drahtfördergurte" is a "gurt"), Dutch or German
  rather than English, as English writes a compound apart;
- its ending ("teilung" is German, "toepassingen" Dutch).

A word that every language has ("transport", "band", a material or a model number) tells nothing.
Each language counts the words that can be in it, the last word twice, because the last word of
a Dutch, German or English phrase is the thing it names and a borrowed word comes before it
("Tri-Flex Gurt" is German). A phrase can be left open ("transportband" is Dutch or German), and
one mostly in another language is in none.

The phrase's language is then, in this order:

1. the language of its pages, when they are all in one, whatever its words say;
2. the language its words tell;
3. the language of most of its pages, among those its words allow;
4. on a tie, the language the site's start page declares, when it is one of the tied ones and the
   words allow it, and never for a phrase without a letter;
5. none.

## Consequences

- No language wins a tie for coming first in some order. A page whose stopwords are as much one
  language's as another's is in none, and its phrases are told by their words.
- On the 22 sites, 69 phrases changed language: 24 from German to Dutch, 19 from German to English,
  9 from Dutch to English, 7 from English to Dutch, 2 from none to English, and 8 to none, which
  are bare material numbers, Spanish words and words that leave it open. Seven more came or went,
  as a phrase in a new language is or is no longer a fragment of a longer one.
- Phrases on pages of one language keep it: 84 of them have words that say otherwise, but half of
  those are product names, too weak a sign to overrule the page.
- A phrase in no language stays a candidate in the review queue, under "Unknown language". A person
  sets its language when accepting it; the language model's acceptance of it is a suggestion only,
  its rejection a decision. The contract is unchanged apart from saying that `languages` can be
  `und`.
- Stored scans are not rewritten. A keyword stored in no language takes the language its words
  tell when a scan, a profile or the review queue reads it, by the map as it is then, and a scan
  that has the phrase in that language already shows it once. A decision or verdict made on such a
  phrase in no language holds once its words tell one, where it applies, as ADR-0008's older
  decisions do. Phrases that went to German on a tie were stored as German; the pages' languages per
  phrase are not stored, so they stay as they are until the site is scanned again.
