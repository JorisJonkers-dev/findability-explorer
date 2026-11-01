-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The competitor definition as versions (the spec's decision 19, 11.11): the text every check of a
-- company is made against, edited in Manage. A saved version never changes; saving makes a new
-- one, and restoring an older one saves its text as a new one, recording which. The newest is the
-- one in use. A check records the version it was made against in its check_version, as before.
-- Version 1 was the check by name and website alone (ADR-0006), with no text of its own; version 2,
-- the target market of ADR-0009 the code held until now, is imported as the first stored version.
-- A new table, so the old pod serves on.
CREATE TABLE IF NOT EXISTS competitor_definitions (
    version bigint PRIMARY KEY CHECK (version BETWEEN 1 AND 100000),
    text text NOT NULL CHECK (char_length(text) BETWEEN 1 AND 8000),
    saved_at timestamptz NOT NULL,
    saved_by text NOT NULL CHECK (char_length(saved_by) BETWEEN 1 AND 200),
    restored_from bigint CHECK (restored_from BETWEEN 1 AND 100000)
);

INSERT INTO competitor_definitions (version, text, saved_at, saved_by)
VALUES (2, 'Tribelt B.V. in Haaksbergen, the Netherlands, is part of the Lesjöfors Group, as is its sister company De Spiraal. It designs and makes metal conveyor belts and their drive elements in its own factory.

Its belts are eyelink (wire mesh) belts, open spiral belts (also called balanced weave), closed spiral belts (also called compound balanced), Tri-Flex stainless steel curve-going belts for spiral freezer and cooler towers, and chain, ladder, plate link, honeycomb, interlaced wire and woven baking belts. Its drive elements are sprockets, tube drums and cage drums.

The belts are for process conveying, where a product is treated while it moves. That is heat in ovens, dryers, furnaces, heat treatment, sintering and glass firing, up to about 1,150 °C. It is cold in spiral freezers and coolers and in cryogenic freezing. It is wet and hygienic steps such as washing, blanching, frying, pasteurising and sterilisation tunnels. And it is heavy, sharp or abrasive loads in recycling, waste and water filtration.

Its customers are industrial food producers of bread and pastry, meat, fish and dairy, potatoes, fruit and vegetables, and meat substitutes. They are the builders of ovens, freezers and lines and the system integrators who build these belts into their machines. And they are industry, such as recycling, waste treatment, water filtration, automotive paint and washing lines, solar and electronics firing furnaces, and pharma and medical sterilisation. Its markets are in Europe, above all the Netherlands and Germany, and English-speaking buyers.

A competitor makes metal conveyor belts (wire, wire mesh, stainless steel link, metal chain or plate belts) for those process uses and sells them to those customers, or has them made and sells them under its own name. A maker anywhere in the world that sells into Europe counts.

These are not competitors. A maker of rubber, textile or steel-cord belting for mining, quarrying and bulk handling is not. A maker of lightweight fabric, PVC or polyurethane belts for packaging, logistics and light conveying is not. A maker of conveyor systems, ovens or freezers that buys its belts rather than making them is a customer or partner, not a competitor. A distributor or trader that does not make belts is not. A maker of wire mesh for fencing, screening, sieving or architecture rather than conveyor belts is not. A directory, marketplace or publisher is not.

A maker of plastic modular belts used in the same food lines, such as in spiral freezers, makes a substitute, not the same product. It is not a competitor unless it also makes metal process belts, and the reason says it makes plastic modular belts.', '2026-10-27T19:46:26Z', 'import')
ON CONFLICT (version) DO NOTHING;
