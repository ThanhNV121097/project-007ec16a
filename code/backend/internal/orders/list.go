package orders

import "github.com/ThanhNV121097/project-007ec16a/backend/internal/data"

// dealerList is the orders collection as the team's list reads it: an order's
// first pair and that pair's shoe are joined in, newest order first, and the
// customer's phone, email, address and note and VNPay's transaction number are
// Private columns. They are served only because Register names Private in the
// served set of the one handler that mounts this list behind auth.Require; a
// request may still filter, range, sort and search only on the public columns,
// and a word is looked for in the order number and the customer's name.
//
// data.ListHandlerServing reads it with the same grammar as every other collection.
var dealerList = &data.Collection{
	Name: "orders",
	From: "orders t LEFT JOIN order_items oi ON oi.order_id = t.id AND oi.line_no = 1" +
		" LEFT JOIN products pr ON pr.id = oi.product_id",
	DefaultOrder: "t.placed_at DESC, t.seq DESC",
	Columns: []data.Column{
		plain("id", data.Text), plain("number", data.Text),
		{Name: "placed_at", Kind: data.Instant,
			Select: "to_char(t.placed_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD\"T\"HH24:MI:SS') || '+07:00'",
			Filter: "t.placed_at"},
		plain("status", data.Text),
		{Name: "product", Kind: data.Text, Select: "coalesce(oi.product_id,'')", Filter: "oi.product_id", Nullable: true},
		{Name: "product_name", Kind: data.Text, Select: "coalesce(pr.name,'')", Filter: "pr.name", Nullable: true},
		{Name: "size", Kind: data.Text, Select: "coalesce(oi.size,'')", Filter: "oi.size", Nullable: true},
		{Name: "qty", Kind: data.Int, Select: "coalesce(oi.qty,0)", Filter: "oi.qty", Nullable: true},
		plain("total", data.Int),
		{Name: "payment", Kind: data.Text, Select: "t.payment_method_id"},
		plain("paid", data.Bool),
		{Name: "delivery", Kind: data.Text, Select: "t.delivery_option_id"},
		plain("carrier", data.Text), plain("tracking", data.Text),
		{Name: "eta", Kind: data.Date, Select: "coalesce(to_char(t.eta,'YYYY-MM-DD'),'')", Filter: "t.eta", Nullable: true},
		plain("customer", data.Text),
		private("phone"), private("email"), private("address"),
		plain("city", data.Text),
		private("note"),
		{Name: "transaction_no", Kind: data.Text, Select: transactionNoExpr, Access: data.Private},
		{Name: "_source", Kind: data.Text, Select: "t.source", Access: data.Bookkeeping},
		{Name: "_invented", Kind: data.List, Select: "t.invented", Access: data.Bookkeeping},
	},
	Search: "strpos(doi_fold(t.number || ' ' || t.customer), {w}) > 0",
}

// plain is a column of orders under its own name.
func plain(name string, kind data.Kind) data.Column {
	return data.Column{Name: name, Kind: kind, Select: "t." + name}
}

// private is a customer's detail: never served, never filtered, never searched.
func private(name string) data.Column {
	c := plain(name, data.Text)
	c.Access = data.Private
	return c
}
