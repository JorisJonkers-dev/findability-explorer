# A language model reviews the term candidates

ADR-0003 grows the terminology map through a review queue in which an administrator accepts or
rejects every phrase the scans find. In practice the scans find hundreds of phrases (663 waited
after the first round), and most of them are clearly in or clearly out. Reviewing them all by hand
is more work than the team will do, so the queue stood still.

**A language model reviews the queue, and a person handles what it is unsure about.** Every
waiting candidate the model has not seen goes to Claude Haiku on an Anthropic key, forty at a
time, with a few map terms of each tier as examples. It is always Haiku on Anthropic's own API,
whatever model product matching uses, because OpenRouter costs more for the same model. The key is
the one Vault delivers in ANTHROPIC_API_KEY, else a stored Anthropic key (ADR-0002) whose latest
test did not fail. For each phrase it says whether to accept it, whether it is sure, why, and the
term it becomes: the text, language, family and tier.

- A sure verdict is recorded as a decision by the language model (`decided_by` is
  `language model`), exactly as an administrator's would be, and can be undone the same way.
- An unsure verdict, or an accepted term the map cannot take, leaves the candidate waiting. The
  Term review page shows the model's reason and its suggestion, which an administrator accepts in
  one step or edits first.
- Each verdict is kept in `term_verdicts`. The model is asked about a candidate once, so an
  undone decision does not come back, and a candidate it skipped counts as unsure.

The review runs in the background every ten minutes and when an administrator asks for it on the
Term review page. Without an Anthropic key, nothing is asked and every phrase waits for a person,
as before.

## Consequences

- This replaces ADR-0003's "the map grows through a review queue, not by itself" for the
  candidates the model is sure about. The queue, the decisions and their undo stay as ADR-0003
  describes.
- What each request used is kept in `term_review_usage` and counts towards the month's tokens on
  the language-model settings page. A review of the whole first queue takes about 70,000 tokens.
  The review does not stop at the monthly budget.
- A wrong decision by the model reaches the map until someone undoes it. The Accepted and Rejected
  tabs name the model as who decided, so its decisions can be checked.
