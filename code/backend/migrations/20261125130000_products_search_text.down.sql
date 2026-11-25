-- The column is derived and nothing else refers to it, so dropping it loses nothing.
ALTER TABLE products DROP COLUMN search_text;
