// Package data serves the shop's records — shoes, lines, categories, reviews,
// orders, shop terms, delivery options, ways to pay, questions and the size
// chart — to anyone, a page at a time, and one record by id.
//
// collections.go is the registry: the ten collections as Go values, a
// hand-written mirror of the frontend's src/data/schema.json (the backend image
// is built from code/backend and cannot embed a file of the frontend; a test
// reads the schema and fails when the two drift). Every column carries an
// access class, and the handlers put ONLY the Public columns in their SELECT:
// a private or bookkeeping column is not in the query, so it cannot be read off
// the wire and no later code can forget to delete it. The registry is also what
// keeps every SQL identifier out of the request's reach: a query string can
// only select an entry of it, never name a column. The same holds for what a
// request may filter, range, sort and search: only Public columns, and only the
// words each collection's Search expression names.
package data

import "strings"

// Access says who may read a column.
type Access int

const (
	// Public columns are served to a caller with no session.
	Public Access = iota
	// Bookkeeping columns (_source, _invented, a shoe's price_listed) are the
	// owner's notes: never served to a caller with no session.
	Bookkeeping
	// Private columns (an order's phone, email, address and note) are a
	// customer's details: never served to a caller with no session.
	Private
)

// Kind is how a filter value is read and bound.
type Kind int

const (
	Text    Kind = iota // exact text; also ids and references
	Int                 // whole number
	Decimal             // number with a fraction
	Bool                // true or false
	Date                // YYYY-MM-DD
	Instant             // RFC 3339 with an offset
	List                // a list of text; a filter means "contains"
)

// cast is the SQL cast that gives one bound filter value its type.
func (k Kind) cast() string {
	switch k {
	case Int:
		return "::bigint"
	case Decimal:
		return "::numeric"
	case Bool:
		return "::boolean"
	case Date:
		return "::date"
	case Instant:
		return "::timestamptz"
	}
	return ""
}

// arrayCast is the SQL cast that gives a bound list of filter values its type.
// A number or a date is bound as text and read by the database, as the single
// value is, so a text-format value never has to be encoded as a number array.
func (k Kind) arrayCast() string {
	switch k {
	case Int:
		return "::bigint[]"
	case Decimal:
		return "::text[]::numeric[]"
	case Bool:
		return "::boolean[]"
	case Date:
		return "::text[]::date[]"
	case Instant:
		return "::timestamptz[]"
	}
	return "::text[]"
}

// ranged says whether a column can be given _from, _to and _ranges.
func (k Kind) ranged() bool {
	return k == Int || k == Decimal || k == Date || k == Instant
}

// Column is one field of a collection.
type Column struct {
	// Name is the field's name in the API, the one the JSON files used.
	Name string
	Kind Kind
	// Select is the SQL expression that produces the value (alias t).
	Select string
	// Filter is the plain column a filter compares; Select when empty.
	Filter string
	Access Access
	// Nullable is a column that can hold no value (a left join, an optional
	// column): a sort puts such records last, whichever way it runs.
	Nullable bool
	// SortExpr is the key a sort uses when it is not the plain column: a
	// shoe's rating is stored 0 when it has none, and sorts as no value.
	SortExpr string
}

func (c Column) filterExpr() string {
	if c.Filter != "" {
		return c.Filter
	}
	return c.Select
}

func (c Column) sortExpr() string {
	if c.SortExpr != "" {
		return c.SortExpr
	}
	return c.filterExpr()
}

// Collection is one of the ten collections the API serves.
type Collection struct {
	// Name is the path segment: /v1/<Name>.
	Name string
	// From is the FROM clause, the table aliased t.
	From    string
	Columns []Column
	// Search is the condition one typed word must meet, written over the
	// searched fields only (never a private column); {w} stands for the word,
	// folded by the database. A record matches when every word meets it.
	Search string
	// Sizes is the condition "at least one of the given EU sizes is listed and
	// not sold out"; {s} stands for the sizes as a text array. Empty for every
	// collection but the shoes.
	Sizes string
}

// public returns the columns a caller with no session may read, in output order.
func (c *Collection) public() []Column {
	var out []Column
	for _, col := range c.Columns {
		if col.Access == Public {
			out = append(out, col)
		}
	}
	return out
}

// selectList is the SELECT list of the public view: each expression under its
// API name.
func (c *Collection) selectList() string {
	parts := make([]string, 0, len(c.Columns))
	for _, col := range c.public() {
		parts = append(parts, col.Select+` AS "`+col.Name+`"`)
	}
	return strings.Join(parts, ", ")
}

// searching sets what a typed word is looked for in.
func (c *Collection) searching(expr string) *Collection {
	c.Search = expr
	return c
}

// sizing sets the EU size condition of the shoes.
func (c *Collection) sizing(expr string) *Collection {
	c.Sizes = expr
	return c
}

// col is a plain column of the table: the API name is the column's name.
func col(name string, kind Kind) Column {
	return Column{Name: name, Kind: kind, Select: "t." + name}
}

// text is a plain text column.
func text(name string) Column { return col(name, Text) }

// nullable marks a column that can hold no value.
func nullable(c Column) Column {
	c.Nullable = true
	return c
}

// ref is a reference served as the id it holds, from the column dbName.
func ref(name, dbName string) Column {
	return Column{Name: name, Kind: Text, Select: "t." + dbName}
}

// optRef is a reference that may be unset: NULL is served as "".
func optRef(name, dbName string) Column {
	return Column{Name: name, Kind: Text, Select: "coalesce(t." + dbName + ",'')", Filter: "t." + dbName, Nullable: true}
}

// day is a date column, served as YYYY-MM-DD in SQL so the answer never
// depends on the server's time zone.
func day(name string) Column {
	return Column{Name: name, Kind: Date, Select: "to_char(t." + name + ",'YYYY-MM-DD')", Filter: "t." + name}
}

// decimal is a numeric column served as a JSON number.
func decimal(name string) Column {
	return Column{Name: name, Kind: Decimal, Select: "t." + name + "::float8", Filter: "t." + name}
}

// bookkeeping are the owner's notes every table carries.
func bookkeeping() []Column {
	return []Column{
		{Name: "_source", Kind: Text, Select: "t.source", Access: Bookkeeping},
		{Name: "_invented", Kind: List, Select: "t.invented", Access: Bookkeeping},
	}
}

func private(name string) Column {
	c := text(name)
	c.Access = Private
	return c
}

func build(name, from string, cols ...Column) *Collection {
	return &Collection{Name: name, From: from, Columns: append(cols, bookkeeping()...)}
}

<<<<<<< refs/heads/feature/checkout-and-placing-an-order-be-v2
// registry lists the nine collections the generic reader serves. Orders are not
// one of them: they hold a customer's details, so internal/orders reads them,
// by the order's own token. The only one with a join is products: a shoe's
// sizes, last-pairs list, sold-out list and stock are derived from the count of
// pairs per size (the view product_stock), never stored on the shoe.
=======
// over is the search condition of a collection whose words are looked for in
// some of its own columns: the columns are joined by a space (so a word never
// straddles two of them), folded, and searched with strpos — which reads % and
// _ as themselves.
func over(columns ...string) string {
	parts := make([]string, len(columns))
	for i, c := range columns {
		parts[i] = "t." + c
	}
	return "strpos(doi_fold(" + strings.Join(parts, " || ' ' || ") + "), {w}) > 0"
}

// registry lists the ten collections. The only one with a join is orders: an
// order's shoe, size and pairs are its first pair line.
>>>>>>> refs/heads/main
var registry = []*Collection{
	build("categories", "categories t",
		text("id"), text("name"), text("blurb"), text("image")).
		searching(over("name", "blurb")),
	build("delivery", "delivery_options t",
		text("id"), text("name"), text("days"), col("fee", Int), text("note")).
		searching(over("name", "days", "note")),
	build("faqs", "faqs t",
		text("id"), text("topic"), text("question"), text("answer")).
		searching(over("topic", "question", "answer")),
	build("lines", "lines t",
<<<<<<< refs/heads/feature/checkout-and-placing-an-order-be-v2
		text("id"), text("name"), text("blurb"), col("since", Int), text("fit_note")),
	build("payments", "payment_methods t",
		text("id"), text("name"), text("note"), col("online", Bool), text("icon")),
	build("products", "products t JOIN product_stock ps ON ps.product_id = t.id",
=======
		text("id"), text("name"), text("blurb"), nullable(col("since", Int)), text("fit_note")).
		searching(over("name", "blurb")),
	build("orders", "orders t LEFT JOIN order_items oi ON oi.order_id = t.id AND oi.line_no = 1",
		text("id"), text("number"),
		Column{Name: "placed_at", Kind: Instant,
			Select: "to_char(t.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD\"T\"HH24:MI:SS') || '+07:00'",
			Filter: "t.placed_at"},
		text("status"),
		Column{Name: "product", Kind: Text, Select: "coalesce(oi.product_id,'')", Filter: "oi.product_id", Nullable: true},
		Column{Name: "size", Kind: Text, Select: "coalesce(oi.size,'')", Filter: "oi.size", Nullable: true},
		Column{Name: "qty", Kind: Int, Select: "coalesce(oi.qty,0)", Filter: "oi.qty", Nullable: true},
		col("total", Int),
		ref("payment", "payment_method_id"), col("paid", Bool), ref("delivery", "delivery_option_id"),
		text("carrier"), text("tracking"),
		Column{Name: "eta", Kind: Date, Select: "coalesce(to_char(t.eta,'YYYY-MM-DD'),'')", Filter: "t.eta", Nullable: true},
		text("customer"),
		private("phone"), private("email"), private("address"),
		text("city"),
		private("note")).
		// The order number and the customer's name, and nothing else: a phone,
		// an e-mail, an address or a note is never searched.
		searching(over("number", "customer")),
	build("payments", "payment_methods t",
		text("id"), text("name"), text("note"), col("online", Bool), text("icon")).
		searching(over("name", "note")),
	build("products", "products t",
>>>>>>> refs/heads/main
		text("id"), text("name"), text("model"),
		optRef("line", "line_id"), optRef("category", "category_id"),
		col("price", Int),
		Column{Name: "price_listed", Kind: Text, Select: "t.price_listed", Access: Bookkeeping},
		text("status"), col("featured", Bool), col("is_new", Bool),
		text("description"), text("code"), text("colourway"), text("colour"), text("swatch"),
		text("upper"), text("lining"), text("sole"), text("fit"), text("care"),
		day("released"),
<<<<<<< refs/heads/feature/checkout-and-placing-an-order-be-v2
		Column{Name: "sizes", Kind: List, Select: "ps.sizes"},
		Column{Name: "sizes_low", Kind: List, Select: "ps.sizes_low"},
		Column{Name: "sizes_out", Kind: List, Select: "ps.sizes_out"},
		Column{Name: "stock", Kind: Int, Select: "ps.stock"},
		decimal("rating"), col("review_count", Int),
		text("image"), text("image_back"), text("image_detail")),
=======
		col("sizes", List), col("sizes_low", List), col("sizes_out", List),
		col("stock", Int), productRating(), col("review_count", Int),
		text("image"), text("image_back"), text("image_detail")).
		// search_text is the shoe's own five fields, folded by the database
		// (migration products_search_text); the line's and the category's name
		// are matched live, so a rename is found at once. search_text is never
		// selected.
		searching("strpos(t.search_text, {w}) > 0" +
			" OR t.line_id IN (SELECT l.id FROM lines l WHERE strpos(doi_fold(l.name), {w}) > 0)" +
			" OR t.category_id IN (SELECT c.id FROM categories c WHERE strpos(doi_fold(c.name), {w}) > 0)").
		// A size is available when the shoe lists it and has not sold it out; a
		// size on its last pairs is still available.
		sizing("EXISTS (SELECT 1 FROM unnest({s}::text[]) AS w(s) WHERE w.s = ANY(t.sizes) AND NOT (w.s = ANY(t.sizes_out)))"),
>>>>>>> refs/heads/main
	build("reviews", "reviews t",
		text("id"), ref("product", "product_id"), text("name"), nullable(col("rating", Int)), day("date"),
		text("size_bought"), text("fit"), text("body"), text("status")).
		searching("strpos(doi_fold(t.name || ' ' || t.body), {w}) > 0" +
			" OR t.product_id IN (SELECT p.id FROM products p WHERE strpos(doi_fold(p.name), {w}) > 0)"),
	build("shop", "shop_terms t",
		text("id"), text("standard_days"), text("dispatch_cutoff"), text("reply_time"),
		decimal("rating"), col("review_count", Int)).
		searching(over("standard_days", "dispatch_cutoff", "reply_time")),
	build("sizes", "size_chart t",
		text("id"), text("eu"), text("uk"), text("us_m"), text("us_w"), nullable(decimal("cm"))).
		searching(over("eu", "uk", "us_m", "us_w")),
}

// productRating is a shoe's average rating. The column is stored 0 when the
// shoe has none, so a sort reads 0 as no value and puts such a shoe last.
func productRating() Column {
	c := decimal("rating")
	c.Nullable = true
	c.SortExpr = "NULLIF(t.rating, 0)"
	return c
}

var byName = func() map[string]*Collection {
	m := make(map[string]*Collection, len(registry))
	for _, c := range registry {
		m[c.Name] = c
	}
	return m
}()
