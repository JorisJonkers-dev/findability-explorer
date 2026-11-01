# A Prompt run can be stopped by hand

A Prompt run asks a few hundred questions and takes a while. Until now it stopped only when it
finished, failed, or reached the month's budget; an administrator who started one by mistake, or saw
it go wrong, could only wait. The app runs on more than one instance, and the one that runs a run is
not always the one that receives a request.

**An administrator stops a running Prompt run, and it keeps the responses it has** (the spec's
decision 5). `stopPromptRun` stores the request on the run (`prompt_runs.stop_requested_at`,
migration `00028_prompt_run_stop.sql`). The instance that runs it reads the request at its next
heartbeat, a quarter of the two minutes after which another instance would take the run over, so a
run stops within about a minute whichever instance received the request. It asks nothing after that,
and ends as stopped, with the failure "Stopped by hand. The responses so far are kept, and a retry
finishes the run." (`domain.StoppedByHand`).

- **How a run ended is told apart.** A stopped run is stopped by hand or by the budget, and the app
  reads which from its failure text. Activity and Runs show every ending as Done, Done with failed
  responses, Stopped by hand, Stopped by the budget, or Failed.
- **A retry finishes a stopped run**: it asks only the questions the run did not get, in the same
  run, with its questions and models.
- **Stop says first what it keeps.** The button is named for the work ("Stop the Prompt run"), and
  the first click says "Stopping keeps the responses so far. You can finish the run later with a
  retry."; the second stops it.
- **The rest of the work stops the same way**, each recorded as stopped by hand and keeping what it
  did: a scan (`stopAnalysis`, which also takes a waiting scan off the queue), every waiting scan
  (`removeQueuedAnalyses`), a rescan (`stopRescan`), the AI readiness check (`stopReadinessCheck`),
  the term review (`stopTermReview`), the review of the competitor candidates
  (`stopCandidateReview`), and the check of the competitor list (`stopCompetitorCheck`). Activity
  offers each of them on the running work. The Recommendation run in the spec has no stop yet, as
  it is not built.
