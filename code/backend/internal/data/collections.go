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
// only select an entry of it, never name a column.
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

// cast is the SQL cast that gives a bound filter value its type.
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
}

func (c Column) filterExpr() string {
	if c.Filter != "" {
		return c.Filter
	}
	return c.Select
}

// Collection is one of the ten collections the API serves.
type Collection struct {
	// Name is the path segment: /v1/<Name>.
	Name string
	// From is the FROM clause, the table aliased t.
	From    string
	Columns []Column
	// TeamOnly puts the collection's LIST behind a dealer session (admin or
	// staff): 401 with no session, 403 for a customer. The record route is not
	// affected.
	TeamOnly bool
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

// col is a plain column of the table: the API name is the column's name.
func col(name string, kind Kind) Column {
	return Column{Name: name, Kind: kind, Select: "t." + name}
}

// text is a plain text column.
func text(name string) Column { return col(name, Text) }

// ref is a reference served as the id it holds, from the column dbName.
func ref(name, dbName string) Column {
	return Column{Name: name, Kind: Text, Select: "t." + dbName}
}

// optRef is a reference that may be unset: NULL is served as "".
func optRef(name, dbName string) Column {
	return Column{Name: name, Kind: Text, Select: "coalesce(t." + dbName + ",'')", Filter: "t." + dbName}
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

// teamOnly marks a collection whose list only the shop's team may read.
func teamOnly(c *Collection) *Collection {
	c.TeamOnly = true
	return c
}

// registry lists the ten collections. The only one with a join is orders: an
// order's shoe, size and pairs are its first pair line.
var registry = []*Collection{
	build("categories", "categories t",
		text("id"), text("name"), text("blurb"), text("image")),
	build("delivery", "delivery_options t",
		text("id"), text("name"), text("days"), col("fee", Int), text("note")),
	build("faqs", "faqs t",
		text("id"), text("topic"), text("question"), text("answer")),
	build("lines", "lines t",
		text("id"), text("name"), text("blurb"), col("since", Int), text("fit_note")),
	teamOnly(build("orders", "orders t LEFT JOIN order_items oi ON oi.order_id = t.id AND oi.line_no = 1",
		text("id"), text("number"),
		Column{Name: "placed_at", Kind: Instant,
			Select: "to_char(t.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD\"T\"HH24:MI:SS') || '+07:00'",
			Filter: "t.placed_at"},
		text("status"),
		Column{Name: "product", Kind: Text, Select: "coalesce(oi.product_id,'')", Filter: "oi.product_id"},
		Column{Name: "size", Kind: Text, Select: "coalesce(oi.size,'')", Filter: "oi.size"},
		Column{Name: "qty", Kind: Int, Select: "coalesce(oi.qty,0)", Filter: "oi.qty"},
		col("total", Int),
		ref("payment", "payment_method_id"), col("paid", Bool), ref("delivery", "delivery_option_id"),
		text("carrier"), text("tracking"),
		Column{Name: "eta", Kind: Date, Select: "coalesce(to_char(t.eta,'YYYY-MM-DD'),'')", Filter: "t.eta"},
		text("customer"),
		private("phone"), private("email"), private("address"),
		text("city"),
		private("note"))),
	build("payments", "payment_methods t",
		text("id"), text("name"), text("note"), col("online", Bool), text("icon")),
	build("products", "products t",
		text("id"), text("name"), text("model"),
		optRef("line", "line_id"), optRef("category", "category_id"),
		col("price", Int),
		Column{Name: "price_listed", Kind: Text, Select: "t.price_listed", Access: Bookkeeping},
		text("status"), col("featured", Bool), col("is_new", Bool),
		text("description"), text("code"), text("colourway"), text("colour"), text("swatch"),
		text("upper"), text("lining"), text("sole"), text("fit"), text("care"),
		day("released"),
		col("sizes", List), col("sizes_low", List), col("sizes_out", List),
		col("stock", Int), decimal("rating"), col("review_count", Int),
		text("image"), text("image_back"), text("image_detail")),
	build("reviews", "reviews t",
		text("id"), ref("product", "product_id"), text("name"), col("rating", Int), day("date"),
		text("size_bought"), text("fit"), text("body"), text("status")),
	build("shop", "shop_terms t",
		text("id"), text("standard_days"), text("dispatch_cutoff"), text("reply_time"),
		decimal("rating"), col("review_count", Int)),
	build("sizes", "size_chart t",
		text("id"), text("eu"), text("uk"), text("us_m"), text("us_w"), decimal("cm")),
}

var byName = func() map[string]*Collection {
	m := make(map[string]*Collection, len(registry))
	for _, c := range registry {
		m[c.Name] = c
	}
	return m
}()
