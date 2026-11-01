# An empty response is a failed response

A model can answer a Prompt run's question with no text at all. A reasoning model that spends its
token limit thinking returns an empty message, and OpenRouter reports it as a success. Until now
such a response was stored like any other: it was read, it named nobody, and it counted towards the
shares as a response that did not name Tribelt. Each empty response pulled AI share of voice down,
and a retry never asked it again, because it had not failed.

**A response with no text is a failed response** (the spec's decision 28). The OpenRouter adapter
returns `domain.ErrEmptyAnswer` ("the model gave no text") when a reply holds nothing but white
space, and the run stores it as failed, with the failure "The model gave no text.".

- **It is never read and never counted.** Like any failed response it has no reading, so it names
  nobody and is left out of every share, and the run ends "Done, with failed responses".
- **A retry asks it again.** `retryPromptRun` asks the failed and missing responses of a run, so an
  empty one gets another try with the same question, model and mode.
- **The responses stored before were fixed in place.** Migration `00027_empty_responses_fail.sql`
  marks every stored response with no text as failed, with the same failure, and deletes its reading
  (the companies it named go with it), so the old runs' figures no longer count them either. No
  text means no character other than white space, as the app tells it (`!~ '\S'`), not only no
  spaces.
- The shares of past runs move when the migration runs, so a chart may change shape at a run with
  many empty responses. They now match what the run actually got.
