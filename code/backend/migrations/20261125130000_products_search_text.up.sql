-- products.search_text: the folded text a shop search looks in.
-- Table design: docs/architecture/erd/search-filters-and-sorting-served-by-the-api.md.
--
-- The shoe's own five searched fields (name, model, colourway, style code,
-- description), joined by one space so a word never straddles two of them and
-- folded by doi_fold, so "Cua Lo" is found by "cua lo" and a search costs one
-- strpos per word instead of a Unicode normalisation per shoe. The database
-- keeps it: it is recomputed in the same statement as every INSERT and UPDATE
-- of a shoe, and it cannot be written. It is never selected. The line's and the
-- category's name are not folded in: they are matched live, so a rename is
-- found at once.
--
-- This rewrites the table once, under an exclusive lock. That is safe here: the
-- server applies migrations before it listens, behind an advisory lock, so no
-- request is waiting on products and no second instance can run this.
ALTER TABLE products ADD COLUMN search_text text NOT NULL GENERATED ALWAYS AS (
    doi_fold(name || ' ' || model || ' ' || colourway || ' ' || code || ' ' || description)
) STORED;

DO $$
BEGIN
    ASSERT doi_fold('Cửa Lò') = 'cua lo', 'doi_fold must fold Cửa Lò to cua lo';
    ASSERT doi_fold('Đại') = 'dai', 'doi_fold must fold Đại to dai';
    ASSERT doi_fold('100%') = '100%', 'doi_fold must leave 100% as it is';
    ASSERT NOT EXISTS (SELECT 1 FROM products WHERE search_text = '' AND btrim(name) <> ''),
        'every shoe with a name must have a search_text';
END $$;
