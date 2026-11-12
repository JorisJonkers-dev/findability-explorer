-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- A response with no text is a failed response (the spec's decision 28): it is not read, not
-- counted, and a retry asks it again. Responses stored before this release were read and counted;
-- they are marked failed here with the failure a run now stores, and their readings go (the
-- companies they name cascade), so the figures no longer count them. No text is no character but
-- white space, as the app tells it; btrim would strip spaces only, not line breaks. The tables hold
-- a few thousand rows, far inside the time limits.
DELETE FROM prompt_extractions e
USING prompt_answers a
WHERE e.answer_id = a.id AND a.failure IS NULL AND a.answer !~ '\S';

UPDATE prompt_answers
SET failure = 'The model gave no text.'
WHERE failure IS NULL AND answer !~ '\S';
