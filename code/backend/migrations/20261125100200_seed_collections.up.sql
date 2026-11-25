-- Seed: every record of the ten JSON files the site read before the database,
-- exactly as they stood after "Free delivery on every order, wording to match",
-- in each file's order (seq follows it).
--
-- Plain INSERTs, never ON CONFLICT: an id that already exists fails this
-- migration loudly and the file rolls back whole, so a half-seeded database
-- cannot exist and an existing record is never merged or overwritten. The
-- migration runner skips an applied file, so a restart inserts nothing.
--
-- Free delivery on every order: delivery holds the one free "standard" option,
-- the shop terms have no threshold, express or return-term column, and the
-- sample orders are totalled as their pairs. None of that is read from a file.
-- This file is literal SQL: it does not read the JSON at run time, because the
-- backend image has no access to the frontend's files.

INSERT INTO categories (id, name, blurb, image, source) VALUES
('women', 'Women', 'Samba OG in three colourways and the Handball Spezial, in EU 36 to 41 1/3.', '/images/ref/category-women.jpg', 'assumed'),
('men', 'Men', 'Gazelle, Adizero and Megaride, in EU 40 to 45 1/3.', '/images/ref/category-men.jpg', 'assumed'),
('kids', 'Kids', 'The Samba OG in grade-school sizes, EU 35 1/2 to 38 2/3.', '/images/ref/category-kids.jpg', 'assumed');

INSERT INTO lines (id, name, blurb, since, fit_note, source, invented) VALUES
('samba', 'Samba', 'Made in 1950 for footballers training on frozen pitches; now the low, narrow everyday shoe.', 1950, 'Narrow through the toe. Take your usual size; go up a third if your feet are wide.', 'web:https://www.featuresneakerboutique.com/search/suggest.json?q=adidas%20samba&resources[type]=product&resources[limit]=6', ARRAY['fit_note']),
('handball-spezial', 'Handball Spezial', 'An indoor handball shoe from 1979, suede and leather on a gum sole.', 1979, 'True to size, with a softer, roomier collar than the Samba.', 'web:https://www.featuresneakerboutique.com/search/suggest.json?q=handball%20spezial&resources[type]=product&resources[limit]=3', ARRAY['fit_note']),
('gazelle', 'Gazelle', 'The 1960s training shoe in suede, worn on terraces ever since.', 1966, 'True to size. The Indoor version sits a little wider at the toe.', 'web:https://www.featuresneakerboutique.com/search/suggest.json?q=adidas%20gazelle&resources[type]=product&resources[limit]=6', ARRAY['since', 'fit_note']),
('adizero', 'Adizero', 'adidas''s light running shoes, from daily trainers to race-day flats.', 2004, 'A snug, race-like fit. Take your usual size, or a third up for long runs.', 'web:https://www.featuresneakerboutique.com/search/suggest.json?q=adidas%20adizero&resources[type]=product&resources[limit]=3', ARRAY['since', 'fit_note']),
('megaride', 'Megaride', 'An early-2000s runner with visible cushioning at the heel, back as a statement shoe.', 2003, 'True to size, with plenty of cushioning under the heel.', 'web:https://www.featuresneakerboutique.com/search/suggest.json?q=adidas%20sl%2072&resources[type]=product&resources[limit]=3', ARRAY['since', 'fit_note']);

INSERT INTO products (id, name, model, line_id, category_id, price, price_listed, status, featured, is_new, description, code, colourway, colour, swatch, upper, lining, sole, fit, care, released, sizes, sizes_low, sizes_out, stock, rating, review_count, image, image_back, image_detail, source, invented) VALUES
('samba-og-w-core-black', 'Samba OG Black Crystal', 'Samba OG', 'samba', 'women', 3290000, 'US$130', 'live', true, true,
 'The 1950s indoor football shoe with its black upper covered in small crystals, white crystal stripes and a dark gum sole. Low and narrow, made for evenings rather than the pitch.',
 'IH9052', 'Core Black / Core Black / Cloud White', 'black', 'black', 'Leather set with small crystals, suede T-toe', 'Synthetic leather', 'Gum rubber cupsole', 'Narrow; take your usual size',
 'Brush the crystals with a soft dry brush and wipe the sole with a damp cloth. Never machine-wash; dry away from the sun.', '2026-11-03',
 ARRAY['36', '36 2/3', '37 1/3', '38', '38 2/3', '39 1/3', '40', '40 2/3', '41 1/3'], ARRAY['37 1/3', '40'], ARRAY['41 1/3'], 18, 4.8, 23,
 '/images/ref/samba-og-w-core-black.jpg', '/images/ref/samba-og-w-core-black-3.jpg', '/images/ref/samba-og-w-core-black-2.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-womens-samba-og-core-black-core-black-cloud-white',
 ARRAY['description', 'upper', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count', 'featured', 'is_new']),
('samba-og-w-crystal-white', 'Samba OG Crystal White', 'Samba OG', 'samba', 'women', 3090000, 'US$120', 'live', false, true,
 'An all-white Samba in beaded leather with serrated stripes, a T-toe and a silver Trefoil on the tongue. Bright enough for a wedding, plain enough for every day after it.',
 'IH9055', 'Crystal White / Core White / Silver Metallic', 'white', 'gainsboro', 'Leather with beaded detail', 'Synthetic', 'Rubber', 'Narrow; take your usual size',
 'Wipe with a damp cloth and a drop of mild soap. Stuff with paper while it dries.', '2026-11-10',
 ARRAY['36', '36 2/3', '37 1/3', '38', '38 2/3', '39 1/3', '40', '40 2/3', '41 1/3'], ARRAY['36', '39 1/3'], '{}'::text[], 24, 4.9, 17,
 '/images/ref/samba-og-w-crystal-white.jpg', '/images/ref/samba-og-w-crystal-white-2.jpg', '/images/ref/samba-og-w-crystal-white-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-womens-samba-og-crystal-white-core-white-silver-metallic',
 ARRAY['fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count', 'is_new']),
('samba-og-w-wonder-quartz', 'Samba OG Wonder Quartz', 'Samba OG', 'samba', 'women', 2790000, 'US$110', 'live', false, true,
 'Soft pink leather, off-white stripes and a gum sole: the Samba in the colour most asked for as a gift this year. Regular fit, low to the ground.',
 'LA6882', 'Wonder Quartz / Off White / Gum', 'pink', 'pink', 'Leather, printed Trefoil', 'Synthetic', 'Gum rubber', 'Regular',
 'Wipe with a damp cloth. Keep the gum sole out of direct sun so it does not yellow.', '2026-11-17',
 ARRAY['36', '36 2/3', '37 1/3', '38', '38 2/3', '39 1/3', '40', '40 2/3', '41 1/3'], ARRAY['38 2/3'], ARRAY['41 1/3'], 31, 4.7, 41,
 '/images/ref/samba-og-w-wonder-quartz.jpg', '/images/ref/samba-og-w-wonder-quartz-2.jpg', '/images/ref/samba-og-w-wonder-quartz-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-womens-samba-og-wonder-quartz-off-white-gum',
 ARRAY['description', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count', 'is_new']),
('handball-spezial-w-earth-strata', 'Handball Spezial Earth Strata', 'Handball Spezial', 'handball-spezial', 'women', 3290000, 'US$130', 'live', false, false,
 'The 1979 handball shoe in brown leather set all over with rhinestones, with gold lettering on the side. The Spezial''s low line and soft collar, dressed up for a night out.',
 'KI2179', 'Earth Strata / Off White / Gold Metallic', 'brown', 'tan', 'Leather with full rhinestone embellishment', 'Textile', 'Rubber', 'Regular; true to size',
 'Brush gently with a soft dry brush. Do not soak; store in its box.', '2026-10-02',
 ARRAY['36', '36 2/3', '37 1/3', '38', '38 2/3', '39 1/3', '40', '40 2/3'], ARRAY['36', '40 2/3'], ARRAY['38'], 9, 4.6, 8,
 '/images/ref/handball-spezial-w-earth-strata.jpg', '/images/ref/handball-spezial-w-earth-strata-2.jpg', '/images/ref/handball-spezial-w-earth-strata-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-womens-handball-spezial-earth-strata-off-white-gold-metallic',
 ARRAY['lining', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']),
('gazelle-indoor-solar-gold', 'Gazelle Indoor Solar Gold', 'Gazelle Indoor', 'gazelle', 'men', 3590000, 'US$140', 'live', false, true,
 'Bad Bunny''s terrace Gazelle in golden suede, with black stripes and heel tab, a doubled tongue and the size label left showing at the heel. Gum rubber underneath.',
 'KI2592', 'Solar Gold / Core Black / Gum', 'yellow', 'goldenrod', 'Suede and leather, flipped T-toe, double tongue', 'Textile', 'Gum rubber', 'Regular; a little wider at the toe',
 'Use a suede brush and eraser; protect with a suede spray before the first rain.', '2026-11-20',
 ARRAY['40', '40 2/3', '41 1/3', '42', '42 2/3', '43 1/3', '44', '44 2/3', '45 1/3'], ARRAY['42', '42 2/3'], ARRAY['40'], 12, 4.8, 14,
 '/images/ref/gazelle-indoor-solar-gold.jpg', '/images/ref/gazelle-indoor-solar-gold-2.jpg', '/images/ref/gazelle-indoor-solar-gold-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-x-bad-bunny-gazelle-indoor-solar-gold-core-black-gum',
 ARRAY['lining', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count', 'is_new']),
('adizero-adios-wonder-blue', 'Adizero Adios Wonder Blue', 'Adizero Adios', 'adizero', 'men', 4090000, 'US$160', 'live', false, false,
 'A light racing shoe made with SATISFY: a stretch woven upper over a full-length Lightstrike Pro midsole that is soft for its weight. For tempo runs and race day.',
 'KH6679', 'Wonder Blue / Pearl Metallic / Wonder Blue', 'blue', 'royalblue', 'One-way stretch woven textile', 'Textile', 'Full-length Lightstrike Pro, rubber outsole', 'Snug, race fit',
 'Hand-wash with cold water and a soft brush; air-dry with the insole out.', '2026-10-14',
 ARRAY['40', '40 2/3', '41 1/3', '42', '42 2/3', '43 1/3', '44', '44 2/3', '45 1/3'], ARRAY['44 2/3'], ARRAY['45 1/3'], 15, 4.7, 11,
 '/images/ref/adizero-adios-wonder-blue.jpg', '/images/ref/adizero-adios-wonder-blue-2.jpg', '/images/ref/adizero-adios-wonder-blue-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-adidas-x-satisfy-adizero-adios-wonder-blue-pearl-metallic-wonder-blue',
 ARRAY['lining', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']);

INSERT INTO products (id, name, model, line_id, category_id, price, price_listed, status, featured, is_new, description, code, colourway, colour, swatch, upper, lining, sole, fit, care, released, sizes, sizes_low, sizes_out, stock, rating, review_count, image, image_back, image_detail, source, invented) VALUES
('adizero-evo-sl-charcoal', 'Adizero Evo SL Charcoal', 'Adizero Evo SL', 'adizero', 'men', 3790000, 'US$150', 'live', false, false,
 'The everyday Adizero, made with Hermanos Koumori in charcoal woven textile with tonal stripes. Lightstrike Pro underfoot for easy runs and long days standing.',
 'KK2048', 'Charcoal / Night Grey / Clear Sky', 'grey', 'dimgray', 'Woven textile, tonal 3-Stripes', 'Textile', 'Lightstrike Pro, rubber outsole', 'Regular; true to size',
 'Hand-wash with cold water; air-dry away from heat.', '2026-09-25',
 ARRAY['40', '40 2/3', '41 1/3', '42', '42 2/3', '43 1/3', '44', '44 2/3'], ARRAY['40', '44 2/3'], '{}'::text[], 20, 4.8, 19,
 '/images/ref/adizero-evo-sl-charcoal.jpg', '/images/ref/adizero-evo-sl-charcoal-2.jpg', '/images/ref/adizero-evo-sl-charcoal-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-adidas-x-hermanos-kuomori-adizero-evo-sl-woven-charcoal-night-grey-clear-sky',
 ARRAY['lining', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']),
('megaride-s2-cloud-white', 'Megaride S2 Cloud White', 'Megaride S2', 'megaride', 'men', 5090000, 'US$200', 'live', false, false,
 'Hellstar''s early-2000s runner: white textile wrapped in sculpted flame overlays, a lace cover and the Megaride cushioning showing at the heel. Loud, and meant to be.',
 'LB5626', 'Cloud White / Bright Red / Core Black', 'white', 'whitesmoke', 'Textile and mixed materials, sculpted flame overlays', 'Textile', 'Megaride cushioning, rubber outsole', 'Regular; true to size',
 'Wipe the overlays with a damp cloth; brush the textile when dry.', '2026-10-21',
 ARRAY['40', '40 2/3', '41 1/3', '42', '42 2/3', '43 1/3', '44'], ARRAY['41 1/3'], ARRAY['44'], 6, 4.6, 9,
 '/images/ref/megaride-s2-cloud-white.jpg', '/images/ref/megaride-s2-cloud-white-2.jpg', '/images/ref/megaride-s2-cloud-white-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-x-hellstar-megaride-s2-footwear-white-brired-core-black',
 ARRAY['lining', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']),
('megaride-bones-halo-green', 'Megaride Bones Halo Green', 'Megaride Bones', 'megaride', 'men', 5890000, 'US$230', 'live', false, false,
 'Willy Chavarria''s Megaride on sculpted, skeleton-like tooling, under a mixed-material upper in pale green and silver. A shoe for someone who wants theirs noticed.',
 'LA6162', 'Halo Green / Silver Metallic / Better Scarlet', 'green', 'palegreen', 'Mixed materials', 'Textile', 'Megaride cushioning, sculpted tooling', 'Regular; true to size',
 'Wipe clean with a damp cloth; keep the silver panels away from solvents.', '2026-09-12',
 ARRAY['40 2/3', '41 1/3', '42', '42 2/3', '43 1/3', '44'], ARRAY['42 2/3', '44'], '{}'::text[], 5, 4.9, 6,
 '/images/ref/megaride-bones-halo-green.jpg', '/images/ref/megaride-bones-halo-green-2.jpg', '/images/ref/megaride-bones-halo-green-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-adidas-x-willy-chavarria-megaride-bones-halo-green-silver-metallic-better-scarlet',
 ARRAY['fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']),
('megaride-ag-xl-core-black', 'Megaride AG XL Core Black', 'Megaride AG XL', 'megaride', 'men', 5590000, '5.769.000 ₫', 'live', false, false,
 'Willy Chavarria''s Megaride AG XL in black on black: ribbed overlays across the upper, a translucent red band of cushioning and the open Megaride tooling under it. "Chavarria" on the tongue.',
 'KJ2041', 'Core Black / Core Black / Red', 'black', 'black', 'Synthetic leather with sculpted ribs, mesh tongue', 'Textile, HD screen-printed sockliner', 'Megaride cushioning, rubber outsole', 'Regular; true to size',
 'Wipe the upper and the red band with a damp cloth; brush the mesh when dry. Keep away from heat.', '2026-08-30',
 ARRAY['41 1/3', '42', '42 2/3', '43 1/3', '44', '44 2/3', '45 1/3'], ARRAY['42', '45 1/3'], ARRAY['44 2/3'], 16, 4.7, 7,
 '/images/ref/megaride-ag-xl-core-black.jpg', '/images/ref/megaride-ag-xl-core-black-2.jpg', '/images/ref/megaride-ag-xl-core-black-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-adidas-x-willy-chavarria-megaride-ag-xl-core-black-core-black-red',
 ARRAY['price', 'description', 'upper', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']),
('samba-og-kids-wonder-mauve', 'Samba OG Kids Wonder Mauve', 'Samba OG Kids', 'samba', 'kids', 2290000, 'US$90', 'live', false, true,
 'The Samba in grade-school sizes: mauve leather, cream stripes and a grey metallic heel, with the same T-toe and rubber sole as the adult pair.',
 'LB3878', 'Wonder Mauve / Cream White / Tech Grey Metallic', 'purple', 'plum', 'Leather, T-toe overlay', 'Synthetic', 'Rubber', 'Regular; leave a thumb''s width to grow',
 'Wipe with a damp cloth. Air-dry, never on a radiator.', '2026-11-14',
 ARRAY['35 1/2', '36', '36 2/3', '37 1/3', '38', '38 2/3'], ARRAY['35 1/2'], '{}'::text[], 14, 4.9, 12,
 '/images/ref/samba-og-kids-wonder-mauve.jpg', '/images/ref/samba-og-kids-wonder-mauve-2.jpg', '/images/ref/samba-og-kids-wonder-mauve-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-kids-samba-og-gs-wonder-mauve-cream-white-tech-grey-metallic',
 ARRAY['lining', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count', 'is_new']),
('samba-og-kids-cloud-white', 'Samba OG Kids Cloud White', 'Samba OG Kids', 'samba', 'kids', 2290000, 'US$90', 'live', false, false,
 'The white-and-black Samba for kids, in leather with a grey suede T-toe and a gum sole. Easy to wipe clean and plain enough for a school day.',
 'KK2348', 'Footwear White / Core Black / Grey One', 'white', 'white', 'Leather, suede T-toe', 'Synthetic', 'Gum rubber', 'Regular; leave a thumb''s width to grow',
 'Wipe with a damp cloth. Air-dry, never on a radiator.', '2026-10-08',
 ARRAY['35 1/2', '36', '36 2/3', '37 1/3', '38', '38 2/3'], ARRAY['38'], ARRAY['38 2/3'], 16, 4.8, 21,
 '/images/ref/samba-og-kids-cloud-white.jpg', '/images/ref/samba-og-kids-cloud-white-2.jpg', '/images/ref/samba-og-kids-cloud-white-3.jpg',
 'web:https://www.featuresneakerboutique.com/products/adidas-originals-kids-samba-og-gs-footwear-white-core-black-grey-one',
 ARRAY['sole', 'lining', 'fit', 'care', 'released', 'sizes', 'sizes_low', 'sizes_out', 'stock', 'rating', 'review_count']);

INSERT INTO reviews (id, product_id, name, rating, date, size_bought, fit, body, status, source) VALUES
('r-01', 'samba-og-w-core-black', 'Phương', 5, '2026-11-21', '38', 'true', 'Wore them to a wedding and then to work on Monday. The crystals catch every light and none have come loose.', 'live', 'assumed'),
('r-02', 'samba-og-w-core-black', 'Thảo', 5, '2026-11-16', '37 1/3', 'small', 'Narrow like every Samba. I went up a third and it is right. Came in two days, box perfect.', 'live', 'assumed'),
('r-03', 'samba-og-w-core-black', 'Vy', 4, '2026-11-08', '39 1/3', 'true', 'Beautiful pair. A little stiff for the first week, then fine.', 'live', 'assumed'),
('r-04', 'samba-og-w-crystal-white', 'Thu Hà', 5, '2026-11-18', '37 1/3', 'true', 'Bought them as a birthday gift for my sister. She has not taken them off.', 'live', 'assumed'),
('r-05', 'samba-og-w-crystal-white', 'Hạnh', 5, '2026-11-12', '38', 'true', 'The beading looks expensive in person. Easy to wipe after a rainy day.', 'live', 'assumed'),
('r-06', 'samba-og-w-crystal-white', 'Quyên', 4, '2026-11-05', '36 2/3', 'small', 'Lovely, but tight at the toe.', 'live', 'assumed'),
('r-07', 'samba-og-w-wonder-quartz', 'Minh Anh', 5, '2026-11-16', '38', 'true', 'The pink is softer than in photos, which I prefer.', 'live', 'assumed'),
('r-08', 'samba-og-w-wonder-quartz', 'Mai Linh', 5, '2026-11-22', '37 1/3', 'true', 'Ordered two pairs, one for me and one for my mother. Both fit with the size chart.', 'live', 'assumed'),
('r-09', 'samba-og-w-wonder-quartz', 'Nhi', 4, '2026-10-29', '39 1/3', 'large', 'Slightly roomy at the heel; a thicker sock solves it. Gum sole is great.', 'live', 'assumed'),
('r-10', 'handball-spezial-w-earth-strata', 'Trang', 5, '2026-11-10', '37 1/3', 'true', 'More comfortable than my Sambas, softer collar. The rhinestones are subtle in daylight.', 'live', 'assumed'),
('r-11', 'handball-spezial-w-earth-strata', 'Yến', 4, '2026-10-24', '38 2/3', 'true', 'Gorgeous colour. Took four days to Cần Thơ, a day longer than promised.', 'live', 'assumed'),
('r-12', 'handball-spezial-w-earth-strata', 'Loan', 5, '2026-10-12', '36 2/3', 'true', 'Exactly the pair in the photos, with the receipt and the box. Will buy here again.', 'live', 'assumed'),
('r-13', 'gazelle-indoor-solar-gold', 'Quang', 5, '2026-11-17', '42', 'true', 'The suede is thick and the colour is even better in the sun. Paid with MoMo in a minute.', 'live', 'assumed'),
('r-14', 'gazelle-indoor-solar-gold', 'Tuấn', 5, '2026-11-23', '43 1/3', 'true', 'Hard to find in Vietnam. Genuine, double tongue and all.', 'live', 'assumed'),
('r-15', 'gazelle-indoor-solar-gold', 'Hiếu', 4, '2026-11-02', '42 2/3', 'large', 'A touch wide; I could have taken 42. Still my favourite pair.', 'live', 'assumed'),
('r-16', 'adizero-adios-wonder-blue', 'Đức', 5, '2026-11-09', '42 2/3', 'small', 'Ran a half marathon in them. Very light, snug fit, go up a third.', 'live', 'assumed'),
('r-17', 'adizero-adios-wonder-blue', 'Sơn', 4, '2026-10-30', '43 1/3', 'true', 'Fast shoe, thin upper. Not for rainy days.', 'live', 'assumed'),
('r-18', 'adizero-adios-wonder-blue', 'Vinh', 5, '2026-10-19', '42', 'true', 'Ordered at night, delivered in Hà Nội the next afternoon.', 'live', 'assumed'),
('r-19', 'adizero-evo-sl-charcoal', 'Bảo', 5, '2026-11-19', '43 1/3', 'true', 'My daily runner now. Soft but not mushy, and the charcoal hides dust.', 'live', 'assumed'),
('r-20', 'adizero-evo-sl-charcoal', 'Thắng', 5, '2026-11-04', '42', 'true', 'I stand all day at work and my feet stopped hurting.', 'live', 'assumed'),
('r-21', 'adizero-evo-sl-charcoal', 'Lâm', 4, '2026-10-21', '44', 'true', 'Great shoe; the laces are a little short.', 'live', 'assumed'),
('r-22', 'megaride-s2-cloud-white', 'Khánh', 5, '2026-11-21', '42', 'true', 'Everyone asks about them. Comfortable for a shoe this loud.', 'live', 'assumed'),
('r-23', 'megaride-s2-cloud-white', 'Phát', 4, '2026-11-01', '43 1/3', 'true', 'Heavier than it looks. Packaging was excellent.', 'live', 'assumed'),
('r-24', 'megaride-s2-cloud-white', 'Duy', 5, '2026-10-25', '41 1/3', 'true', 'Real pair, real receipt. The shop answered my size question within the hour.', 'live', 'assumed'),
('r-25', 'megaride-bones-halo-green', 'Long', 5, '2026-11-25', '43 1/3', 'true', 'Arrived the morning after I ordered. The sole is a sculpture.', 'live', 'assumed'),
('r-26', 'megaride-bones-halo-green', 'Nhật', 5, '2026-10-30', '42', 'true', 'Not for everyone, which is the point. Fits like my other adidas.', 'live', 'assumed'),
('r-27', 'megaride-bones-halo-green', 'Toàn', 4, '2026-10-16', '42 2/3', 'large', 'A little long; I added an insole. Colour is exactly as shown.', 'live', 'assumed'),
('r-28', 'samba-og-kids-wonder-mauve', 'Ngọc', 5, '2026-11-20', '36', 'true', 'My daughter''s first adidas. I measured her foot with the chart and the size was right.', 'live', 'assumed'),
('r-29', 'samba-og-kids-wonder-mauve', 'Hiền', 5, '2026-11-11', '35 1/2', 'true', 'Sturdy and easy to clean after school.', 'live', 'assumed'),
('r-30', 'samba-og-kids-wonder-mauve', 'Tâm', 4, '2026-10-28', '37 1/3', 'small', 'Runs a little small, like the adult Samba.', 'live', 'assumed'),
('r-31', 'samba-og-kids-cloud-white', 'Hương', 5, '2026-11-24', '37 1/3', 'true', 'Matching pair with mine. My son is very proud.', 'live', 'assumed'),
('r-32', 'samba-og-kids-cloud-white', 'Thủy', 5, '2026-11-06', '36 2/3', 'true', 'White stays white with a damp cloth. Good value for a genuine pair.', 'live', 'assumed'),
('r-33', 'samba-og-kids-cloud-white', 'Kiên', 4, '2026-10-15', '38', 'true', 'Took a week to soften, now worn every day.', 'live', 'assumed'),
('r-34', 'megaride-ag-xl-core-black', 'Phong', 5, '2026-11-19', '43 1/3', 'true', 'All black until you see the red band under the light. Fits like my Adizero.', 'live', 'assumed'),
('r-35', 'megaride-ag-xl-core-black', 'Đạt', 5, '2026-11-02', '42', 'true', 'Heavier than it looks, but soft all day. Came double-boxed.', 'live', 'assumed'),
('r-36', 'megaride-ag-xl-core-black', 'Quân', 4, '2026-10-18', '44', 'large', 'Roomy at the toe; a third down would have been right.', 'live', 'assumed');

INSERT INTO shop_terms (id, standard_days, dispatch_cutoff, reply_time, rating, review_count, source) VALUES
('shop', '2–4', '15:00', 'the same working day', 4.8, 181, 'assumed');

INSERT INTO delivery_options (id, name, days, fee, note, source) VALUES
('standard', 'Standard', '2–4 working days', 0, 'Nationwide with GHN or GHTK, tracked from our door to yours.', 'assumed');

INSERT INTO payment_methods (id, name, note, online, icon, source) VALUES
('cod', 'Cash on delivery', 'Pay the courier in cash when the box arrives. You may open it and check the pair first.', false, 'cash', 'assumed'),
('bank', 'Bank transfer (VietQR)', 'Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.', true, 'qr', 'assumed'),
('vnpay', 'VNPay — ATM, Visa, Mastercard', 'Pay by card or banking app on VNPay''s secure page.', true, 'card', 'assumed'),
('momo', 'MoMo wallet', 'Confirm the payment in the MoMo app; the order is marked paid at once.', true, 'wallet', 'assumed');

INSERT INTO faqs (id, topic, question, answer, source) VALUES
('f-delivery-time', 'shipping', 'How long does delivery take?', 'Two to four working days anywhere in Vietnam by standard delivery.', 'assumed'),
('f-delivery-where', 'shipping', 'Do you deliver to every province?', 'Yes. GHN and GHTK reach all 63 provinces; island districts can take one or two days longer.', 'assumed'),
('f-delivery-change', 'shipping', 'Can I change the address after ordering?', 'Yes, until the order is packed. Send us the order number and the new address through the contact form.', 'assumed'),
('f-delivery-gift', 'shipping', 'Can you send a pair as a gift?', 'Yes. Put the recipient''s name and address at checkout and add "gift" in the note; we leave the price out of the box.', 'assumed'),
('f-returns-size', 'returns', 'The size is wrong. What do I do?', 'No exchanges are accepted, so please check the size chart before you order. If you are unsure of a size, ask us before you order.', 'owner'),
('f-returns-wrong', 'returns', 'My delivery is wrong or damaged.', 'Contact us with your order number: call, message us on Zalo or send the contact form.', 'assumed'),
('f-sizing-small', 'sizing', 'Do adidas shoes run small?', 'Most run true to length. The Samba is narrow, so wide feet usually go up a third; each shoe''s page says how it fits.', 'assumed'),
('f-sizing-between', 'sizing', 'I am between two sizes.', 'Take the larger one for running shoes and for wide feet, the smaller for a leather Samba that will soften. If you are unsure, ask us before you order.', 'assumed'),
('f-sizing-kids', 'sizing', 'How do I measure a child''s foot?', 'Measure standing, in the afternoon, and add about a centimetre to grow into: a thumb''s width in front of the longest toe.', 'assumed'),
('f-payment-how', 'payment', 'How do I pay?', 'Online through VNPay, by card, domestic ATM card or QR.', 'assumed'),
('f-payment-genuine', 'payment', 'Are the shoes genuine?', 'Every pair comes in its adidas box with its tags and our receipt.', 'assumed'),
('f-orders-number', 'orders', 'Where is my order number?', 'On the page you saw after paying, starting with DOI-.', 'assumed');

INSERT INTO size_chart (id, eu, uk, us_m, us_w, cm, source, invented) VALUES
('eu-35-1-2', '35 1/2', '3', '3.5', '4.5', 21.6, 'web:https://www.size.ly/size-chart/adidas', ARRAY['cm']),
('eu-36', '36', '3.5', '4', '5', 22.1, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-36-2-3', '36 2/3', '4', '4.5', '5.5', 22.5, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-37-1-3', '37 1/3', '4.5', '5', '6', 22.9, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-38', '38', '5', '5.5', '6.5', 23.3, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-38-2-3', '38 2/3', '5.5', '6', '7', 23.8, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-39-1-3', '39 1/3', '6', '6.5', '7.5', 24.2, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-40', '40', '6.5', '7', '8', 24.6, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-40-2-3', '40 2/3', '7', '7.5', '8.5', 25.0, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-41-1-3', '41 1/3', '7.5', '8', '9', 25.5, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-42', '42', '8', '8.5', '9.5', 25.9, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-42-2-3', '42 2/3', '8.5', '9', '10', 26.3, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-43-1-3', '43 1/3', '9', '9.5', '10.5', 26.7, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-44', '44', '9.5', '10', '11', 27.1, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-44-2-3', '44 2/3', '10', '10.5', '11.5', 27.6, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-45-1-3', '45 1/3', '10.5', '11', '12', 28.0, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]),
('eu-46', '46', '11', '11.5', '12.5', 28.4, 'web:https://www.size.ly/size-chart/adidas', '{}'::text[]);

-- The 14 sample orders. placed_at is the file's zone-less time read as Vietnam
-- time. delivery is always the free standard option, so delivery_fee is 0 and
-- every total is its pairs' price. A pair line follows each order below.
INSERT INTO orders (id, number, placed_at, status, payment_method_id, paid, delivery_option_id, delivery_fee, carrier, tracking, eta, customer, phone, email, address, city, note, total, is_sample, source) VALUES
('doi-24817', 'DOI-24817', '2026-11-23T16:05:00+07:00', 'shipped', 'momo', true, 'standard', 0, 'GHN', 'GHN-PT27XM', '2026-11-26', 'Hoàng Nam', '0938 410 552', '', '42 Example Street, Hải Châu', 'Đà Nẵng', 'Gift — no price inside the box.', 3090000, true, 'assumed'),
('doi-24816', 'DOI-24816', '2026-11-25T08:40:00+07:00', 'placed', 'cod', false, 'standard', 0, '', '', '2026-11-28', 'Trang', '0977 302 118', '', '7 Example Lane', 'Biên Hòa', '', 3290000, true, 'assumed'),
('doi-24815', 'DOI-24815', '2026-11-24T10:12:00+07:00', 'confirmed', 'vnpay', true, 'standard', 0, 'GHN', '', '2026-11-25', 'Long', '0912 774 630', '', '15 Example Road, Ba Đình', 'Hà Nội', '', 5890000, true, 'assumed'),
('doi-24814', 'DOI-24814', '2026-11-23T19:30:00+07:00', 'packed', 'cod', false, 'standard', 0, 'GHTK', '', '2026-11-27', 'Hương', '0905 661 207', '', '3 Example Street', 'Vũng Tàu', '', 2290000, true, 'assumed'),
('doi-24813', 'DOI-24813', '2026-11-22T11:02:00+07:00', 'shipped', 'momo', true, 'standard', 0, 'GHN', 'GHN-KQ81DA', '2026-11-25', 'Tuấn', '0983 215 904', '', '21 Example Street, Cầu Giấy', 'Hà Nội', '', 3590000, true, 'assumed'),
('doi-24812', 'DOI-24812', '2026-11-21T14:48:00+07:00', 'shipped', 'bank', true, 'standard', 0, 'GHTK', 'GHTK-5520193', '2026-11-26', 'Mai Linh', '0934 880 312', '', '9 Example Avenue', 'Nha Trang', '', 5580000, true, 'assumed'),
('doi-24811', 'DOI-24811', '2026-11-20T09:20:00+07:00', 'cancelled', 'cod', false, 'standard', 0, '', '', NULL, 'Đức', '0919 040 776', '', '5 Example Street', 'Huế', 'Customer cancelled by phone; ordered 43 1/3 instead in store.', 4090000, true, 'assumed'),
('doi-24810', 'DOI-24810', '2026-11-19T21:15:00+07:00', 'delivered', 'vnpay', true, 'standard', 0, 'GHN', 'GHN-MW55RB', '2026-11-20', 'Phương', '0908 552 671', '', '88 Example Street, Quận 3', 'TP. Hồ Chí Minh', '', 3290000, true, 'assumed'),
('doi-24809', 'DOI-24809', '2026-11-18T13:44:00+07:00', 'delivered', 'momo', true, 'standard', 0, 'GHN', 'GHN-HC09TT', '2026-11-21', 'Khánh', '0937 118 249', '', '12 Example Road, Bình Thạnh', 'TP. Hồ Chí Minh', '', 5090000, true, 'assumed'),
('doi-24808', 'DOI-24808', '2026-11-17T08:05:00+07:00', 'delivered', 'cod', true, 'standard', 0, 'GHTK', 'GHTK-5518840', '2026-11-20', 'Ngọc', '0915 330 482', '', '30 Example Street', 'Hải Phòng', '', 2290000, true, 'assumed'),
('doi-24807', 'DOI-24807', '2026-11-15T17:36:00+07:00', 'delivered', 'bank', true, 'standard', 0, 'GHN', 'GHN-AZ71PE', '2026-11-19', 'Bảo', '0939 602 115', '', '6 Example Street, Ninh Kiều', 'Cần Thơ', '', 3790000, true, 'assumed'),
('doi-24806', 'DOI-24806', '2026-11-14T10:58:00+07:00', 'delivered', 'vnpay', true, 'standard', 0, 'GHN', 'GHN-RR42LN', '2026-11-15', 'Thu Hà', '0904 771 390', '', '18 Example Lane, Hoàn Kiếm', 'Hà Nội', '', 3090000, true, 'assumed'),
('doi-24805', 'DOI-24805', '2026-11-13T15:21:00+07:00', 'delivered', 'momo', true, 'standard', 0, 'GHTK', 'GHTK-5507712', '2026-11-16', 'Quang', '0935 248 016', '', '27 Example Street, Sơn Trà', 'Đà Nẵng', '', 3590000, true, 'assumed'),
('doi-24804', 'DOI-24804', '2026-11-12T09:14:00+07:00', 'delivered', 'cod', true, 'standard', 0, 'GHN', 'GHN-LJ8K4Q', '2026-11-15', 'Minh Anh', '0903 118 274', '', '64 Example Street, Đống Đa', 'Hà Nội', '', 2790000, true, 'assumed');

INSERT INTO order_items (order_id, line_no, product_id, size, qty, unit_price) VALUES
('doi-24817', 1, 'samba-og-w-crystal-white', '39 1/3', 1, 3090000),
('doi-24816', 1, 'handball-spezial-w-earth-strata', '37 1/3', 1, 3290000),
('doi-24815', 1, 'megaride-bones-halo-green', '43 1/3', 1, 5890000),
('doi-24814', 1, 'samba-og-kids-cloud-white', '37 1/3', 1, 2290000),
('doi-24813', 1, 'gazelle-indoor-solar-gold', '43 1/3', 1, 3590000),
('doi-24812', 1, 'samba-og-w-wonder-quartz', '37 1/3', 2, 2790000),
('doi-24811', 1, 'adizero-adios-wonder-blue', '42 2/3', 1, 4090000),
('doi-24810', 1, 'samba-og-w-core-black', '38', 1, 3290000),
('doi-24809', 1, 'megaride-s2-cloud-white', '42', 1, 5090000),
('doi-24808', 1, 'samba-og-kids-wonder-mauve', '36', 1, 2290000),
('doi-24807', 1, 'adizero-evo-sl-charcoal', '43 1/3', 1, 3790000),
('doi-24806', 1, 'samba-og-w-crystal-white', '37 1/3', 1, 3090000),
('doi-24805', 1, 'gazelle-indoor-solar-gold', '42', 1, 3590000),
('doi-24804', 1, 'samba-og-w-wonder-quartz', '38', 1, 2790000);

-- The migration checks its own work. Each count is the number of records of
-- the file above; a stale file that still held an express order or an old total
-- fails here, loudly, instead of being edited silently.
DO $$
BEGIN
    ASSERT (SELECT count(*) FROM categories) = 3, 'categories: expected 3 records';
    ASSERT (SELECT count(*) FROM lines) = 5, 'lines: expected 5 records';
    ASSERT (SELECT count(*) FROM products) = 12, 'products: expected 12 records';
    ASSERT (SELECT count(*) FROM reviews) = 36, 'reviews: expected 36 records';
    ASSERT (SELECT count(*) FROM shop_terms) = 1, 'shop_terms: expected 1 record';
    ASSERT (SELECT count(*) FROM delivery_options) = 1, 'delivery_options: expected 1 record';
    ASSERT (SELECT count(*) FROM payment_methods) = 4, 'payment_methods: expected 4 records';
    ASSERT (SELECT count(*) FROM faqs) = 12, 'faqs: expected 12 records';
    ASSERT (SELECT count(*) FROM size_chart) = 17, 'size_chart: expected 17 records';
    ASSERT (SELECT count(*) FROM orders) = 14, 'orders: expected 14 records';
    ASSERT (SELECT count(*) FROM order_items) = 14, 'order_items: expected 14 records';
    ASSERT NOT EXISTS (
        SELECT 1 FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.unit_price <> p.price
    ), 'a sample order line is not priced at its shoe''s price';
    ASSERT NOT EXISTS (
        SELECT 1 FROM orders o JOIN order_items oi ON oi.order_id = o.id
        WHERE o.total <> oi.unit_price * oi.qty + o.delivery_fee
    ), 'a sample order total is not its pairs'' price plus its delivery fee';
END $$;
