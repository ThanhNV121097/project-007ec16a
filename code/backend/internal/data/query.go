package data

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultLimit = 12
	maxLimit     = 100
	maxPage      = 2147483647

	// maxQueryRunes bounds q, so a search is never longer than a person types.
	maxQueryRunes = 100
	// maxRanges bounds one <field>_ranges list.
	maxRanges = 20

	badParameters = "Some parameters are not valid."
)

var (
	wholeNumber  = regexp.MustCompile(`^-?[0-9]{1,18}$`)
	decimalNum   = regexp.MustCompile(`^-?[0-9]{1,18}(\.[0-9]{1,6})?$`)
	rangeNumber  = regexp.MustCompile(`^-?[0-9]{1,40}(\.[0-9]{1,6})?$`)
	digitsOnly   = regexp.MustCompile(`^[0-9]+$`)
	idSlug       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	maxIDLength  = 100
	dateLayout   = "2006-01-02"
	instantUsage = "such as 2026-11-23T16:05:00+07:00"
)

// validID says whether a path segment can be a record's id. It is checked
// before any query, so a segment that cannot be an id never reaches the
// database.
func validID(id string) bool {
	return len(id) <= maxIDLength && idSlug.MatchString(id)
}

// filter is one "any of" condition: the registry's column and what is bound.
// arg is one value of the column's kind, or — when many is set — a list of
// them: category=women is "= $1", category=women,kids is "= ANY($1)".
type filter struct {
	col  Column
	arg  any
	many bool
}

// sql is the condition, with ph the placeholder of its value.
func (f filter) sql(ph string) string {
	expr := f.col.filterExpr()
	switch {
	case f.col.Kind == List && f.many:
		// Any element in common.
		return expr + " && " + ph + "::text[]"
	case f.col.Kind == List:
		return ph + " = ANY(" + expr + ")"
	case f.many:
		return expr + " = ANY(" + ph + f.col.Kind.arrayCast() + ")"
	}
	return expr + " = " + ph + f.col.Kind.cast()
}

// edge is one end of a range: lower bounds are included, upper bounds too.
type edge struct {
	upper bool
	val   string
}

// rangeFilter is "the column is inside any of these ranges"; each range is the
// edges it has, all of which must hold. <field>_from and <field>_to are a
// single range with one edge.
type rangeFilter struct {
	col  Column
	alts [][]edge
}

// sql is the condition; next binds one value and returns its placeholder.
func (r rangeFilter) sql(next func(any) string) string {
	alts := make([]string, 0, len(r.alts))
	for _, alt := range r.alts {
		parts := make([]string, 0, len(alt))
		for _, e := range alt {
			parts = append(parts, r.col.edgeSQL(e.upper, next(e.val)))
		}
		alts = append(alts, "("+strings.Join(parts, " AND ")+")")
	}
	if len(alts) == 1 {
		return alts[0]
	}
	return "(" + strings.Join(alts, " OR ") + ")"
}

// edgeSQL compares the column with one bound end. An order's moment is ranged
// by calendar days in Vietnam, so a "to" day holds the whole of that day.
func (c Column) edgeSQL(upper bool, ph string) string {
	expr := c.filterExpr()
	switch c.Kind {
	case Instant:
		if upper {
			return expr + " < ((" + ph + "::date + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')"
		}
		return expr + " >= (" + ph + "::date::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')"
	case Date:
		if upper {
			return expr + " <= " + ph + "::date"
		}
		return expr + " >= " + ph + "::date"
	}
	if upper {
		return expr + " <= " + ph + "::numeric"
	}
	return expr + " >= " + ph + "::numeric"
}

// sortKey is the one field a list is ordered by.
type sortKey struct {
	col  Column
	desc bool
}

// sql is the key written for ORDER BY. A record with no value comes last
// whichever way the key runs, but only a column that can hold none says so: a
// NOT NULL column carries no NULLS clause, which would stop an index on it
// from serving the order. Text sorts by its folded first hundred characters, so
// "Đại" sorts with the D's whatever collation the database was made with.
func (s sortKey) sql() string {
	dir := " ASC"
	if s.desc {
		dir = " DESC"
	}
	if s.col.Nullable {
		dir += " NULLS LAST"
	}
	expr := s.col.sortExpr()
	if s.col.Kind == Text {
		return "doi_fold(left(" + expr + ", 100))" + dir + ", " + expr + dir
	}
	return expr + dir
}

// listRequest is a validated GET /v1/<collection>.
type listRequest struct {
	page    int64
	limit   int64
	filters []filter
	ranges  []rangeFilter
	// sizes are the EU sizes of a shoe filter: one must be available.
	sizes []string
	// words are the typed words, each of which must be found.
	words []string
	// sort is nil for the seed's order.
	sort *sortKey
<<<<<<< refs/heads/feature/manage-shoes-lines-categories-reviews-and-orders-be
	// dealer is true for a session of the shop's team: the list is not narrowed
	// to what is on sale and the dealer's columns are served.
	dealer bool
=======
	// served are the access classes the handler selects; none means Public.
	served []Access
>>>>>>> refs/heads/main
}

// firstValue is the first value of a repeated name.
func firstValue(q url.Values, key string) (string, bool) {
	vals, ok := q[key]
	if !ok || len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// parseList reads the query string of a list request. It returns every fault
// at once, page and limit first; with any fault nothing is returned.
//
// The keys it reads are page, limit, q, sort, size (shoes only), a field's
// name (any of), and a field's name with _from, _to or _ranges. A name that is
// none of these is ignored, so an address a person edited still answers. A
// filter, range or sort on a column that is not public is a fault, whatever its
// value: the answer to it would reveal what the column hides.
func parseList(c *Collection, q url.Values) (listRequest, map[string]string) {
	faults := map[string]string{}
	req := listRequest{page: 1, limit: defaultLimit}

	if v, ok := firstValue(q, "page"); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if !digitsOnly.MatchString(v) || err != nil || n < 1 || n > maxPage {
			faults["page"] = "page must be a whole number from 1."
		} else {
			req.page = int64(n)
		}
	}
	if v, ok := firstValue(q, "limit"); ok {
		req.limit = parseLimit(v, faults)
	}

	readWords(q, &req, faults)
	readSort(c, q, &req, faults)
	if c.Sizes != "" {
		readSizes(q, &req, faults)
	}
	for _, column := range c.Columns {
		if c.Sizes != "" && column.Name == "size" {
			continue
		}
		readFilter(column, q, &req, faults)
		readRanges(column, q, &req, faults)
	}
	if len(faults) > 0 {
		return listRequest{}, faults
	}
	return req, nil
}

// parseLimit reads limit: digits only, at least 1, and above 100 served as 100
// (a number too long to parse is above 100 too).
func parseLimit(v string, faults map[string]string) int64 {
	if !digitsOnly.MatchString(v) {
		faults["limit"] = "limit must be a whole number from 1."
		return defaultLimit
	}
	n, err := strconv.ParseUint(v, 10, 64)
	switch {
	case err != nil || n > maxLimit:
		return maxLimit
	case n < 1:
		faults["limit"] = "limit must be a whole number from 1."
		return defaultLimit
	}
	return int64(n)
}

// readableText says whether PostgreSQL can hold v as text. It refuses a NUL
// and any byte sequence that is not UTF-8, and a statement that is handed one
// fails — which would be answered 500 for what is the caller's typo.
func readableText(v string) bool {
	return utf8.ValidString(v) && !strings.ContainsRune(v, 0)
}

func notAllowed(key string) string {
	return key + " holds a character that is not allowed."
}

// readWords reads q: trimmed, empty means no search, at most 100 characters,
// split on white space with exact repeats dropped.
func readWords(q url.Values, req *listRequest, faults map[string]string) {
	raw, ok := firstValue(q, "q")
	if !ok {
		return
	}
	if !readableText(raw) {
		faults["q"] = notAllowed("q")
		return
	}
	trimmed := strings.TrimSpace(raw)
	if utf8.RuneCountInString(trimmed) > maxQueryRunes {
		faults["q"] = "q can have at most 100 characters."
		return
	}
	seen := map[string]bool{}
	for _, w := range strings.Fields(trimmed) {
		if !seen[w] {
			seen[w] = true
			req.words = append(req.words, w)
		}
	}
}

// readSort reads sort: one public field, written -field for descending.
func readSort(c *Collection, q url.Values, req *listRequest, faults map[string]string) {
	raw, ok := firstValue(q, "sort")
	if !ok {
		return
	}
	if !readableText(raw) {
		faults["sort"] = notAllowed("sort")
		return
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return
	}
	name, desc := strings.TrimPrefix(s, "-"), strings.HasPrefix(s, "-")
	for _, column := range c.Columns {
		if column.Name != name {
			continue
		}
		if column.Access != Public {
			faults["sort"] = name + " cannot be sorted."
			return
		}
		req.sort = &sortKey{col: column, desc: desc}
		return
	}
	faults["sort"] = "sort must be a field of this list, such as price or -price."
}

// readSizes reads size on a collection that has a size condition (the shoes).
func readSizes(q url.Values, req *listRequest, faults map[string]string) {
	raw, ok := firstValue(q, "size")
	if !ok {
		return
	}
	items, fault := listItems("size", raw)
	if fault != "" {
		faults["size"] = fault
		return
	}
	req.sizes = items
}

// listItems splits a value on commas: a comma is always a separator. Every item
// is trimmed, empty items and repeats are dropped; nothing left means the key
// is ignored.
func listItems(key, raw string) ([]string, string) {
	if !readableText(raw) {
		return nil, notAllowed(key)
	}
	var items []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	return items, ""
}

// readFilter reads <field>=a,b.
func readFilter(column Column, q url.Values, req *listRequest, faults map[string]string) {
	raw, ok := firstValue(q, column.Name)
	if !ok {
		return
	}
	if column.Access != Public {
		faults[column.Name] = column.Name + " cannot be used as a filter."
		return
	}
	items, fault := listItems(column.Name, raw)
	if fault != "" {
		faults[column.Name] = fault
		return
	}
	if len(items) == 0 {
		return
	}
	arg, many, fault := bindItems(column, items)
	if fault != "" {
		faults[column.Name] = fault
		return
	}
	req.filters = append(req.filters, filter{col: column, arg: arg, many: many})
}

// readRanges reads <field>_from, <field>_to and <field>_ranges.
func readRanges(column Column, q url.Values, req *listRequest, faults map[string]string) {
	for _, suffix := range []string{"_from", "_to", "_ranges"} {
		key := column.Name + suffix
		raw, ok := firstValue(q, key)
		if !ok {
			continue
		}
		if column.Access != Public {
			faults[key] = key + " cannot be used as a filter."
			continue
		}
		if !column.Kind.ranged() {
			continue
		}
		if !readableText(raw) {
			faults[key] = notAllowed(key)
			continue
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if suffix == "_ranges" {
			alts, fault := parseRanges(column, key, raw)
			if fault != "" {
				faults[key] = fault
				continue
			}
			req.ranges = append(req.ranges, rangeFilter{col: column, alts: alts})
			continue
		}
		if fault := checkEdge(column, key, raw); fault != "" {
			faults[key] = fault
			continue
		}
		req.ranges = append(req.ranges, rangeFilter{col: column, alts: [][]edge{{{upper: suffix == "_to", val: raw}}}})
	}
}

// checkEdge says why v cannot be one end of a range of the column, or "".
func checkEdge(column Column, key, v string) string {
	if column.Kind == Date || column.Kind == Instant {
		// Go reads year 0000; PostgreSQL has no such year.
		if d, err := time.Parse(dateLayout, v); err != nil || d.Year() < 1 {
			return key + " must be a date written YYYY-MM-DD."
		}
		return ""
	}
	if !rangeNumber.MatchString(v) {
		return key + " must be a number."
	}
	return ""
}

// parseRanges reads from..to,from..to: either end may be left empty, not both.
// The first ".." of a part splits it.
func parseRanges(column Column, key, raw string) ([][]edge, string) {
	example := "..2999999,4000000.."
	if column.Kind == Date || column.Kind == Instant {
		example = "..2026-03-31,2026-07-01.."
	}
	parts := strings.Split(raw, ",")
	if len(parts) > maxRanges {
		return nil, key + " can hold at most 20 ranges."
	}
	alts := make([][]edge, 0, len(parts))
	for _, part := range parts {
		lo, hi, found := strings.Cut(strings.TrimSpace(part), "..")
		lo, hi = strings.TrimSpace(lo), strings.TrimSpace(hi)
		if !found || (lo == "" && hi == "") {
			return nil, key + " must be ranges written from..to, such as " + example
		}
		var alt []edge
		for _, end := range []edge{{upper: false, val: lo}, {upper: true, val: hi}} {
			if end.val == "" {
				continue
			}
			if fault := checkEdge(column, key, end.val); fault != "" {
				return nil, fault
			}
			alt = append(alt, end)
		}
		alts = append(alts, alt)
	}
	return alts, ""
}

// bindItems turns the items of a filter into what is bound to its placeholder
// — one value, or a list of them — or the sentence that says why one cannot be
// read as the field's kind.
func bindItems(column Column, items []string) (any, bool, string) {
	if len(items) == 1 {
		arg, fault := bindOne(column, items[0])
		return arg, false, fault
	}
	var arg any
	var fault string
	switch column.Kind {
	case Int:
		arg, fault = collect[int64](column, items)
	case Bool:
		arg, fault = collect[bool](column, items)
	case Instant:
		arg, fault = collect[time.Time](column, items)
	default:
		arg, fault = collect[string](column, items)
	}
	if fault != "" {
		return nil, false, fault
	}
	return arg, true, ""
}

func collect[T any](column Column, items []string) (any, string) {
	out := make([]T, 0, len(items))
	for _, item := range items {
		v, fault := bindOne(column, item)
		if fault != "" {
			return nil, fault
		}
		out = append(out, v.(T))
	}
	return out, ""
}

// bindOne reads one item as the column's kind, or returns the sentence that
// says why it cannot be. A value PostgreSQL would refuse is refused here,
// before any statement is sent.
func bindOne(col Column, v string) (any, string) {
	switch col.Kind {
	case Int:
		n, err := strconv.ParseInt(v, 10, 64)
		if !wholeNumber.MatchString(v) || err != nil {
			return nil, col.Name + " must be a whole number."
		}
		return n, ""
	case Decimal:
		if !decimalNum.MatchString(v) {
			return nil, col.Name + " must be a number."
		}
		return v, ""
	case Bool:
		if v != "true" && v != "false" {
			return nil, col.Name + " must be true or false."
		}
		return v == "true", ""
	case Date:
		// Go reads year 0000; PostgreSQL has no such year.
		if d, err := time.Parse(dateLayout, v); err != nil || d.Year() < 1 {
			return nil, col.Name + " must be a date written YYYY-MM-DD."
		}
		return v, ""
	case Instant:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil || t.Year() < 1 {
			return nil, col.Name + " must be a date and time " + instantUsage + "."
		}
		return t, ""
	}
	return v, ""
}

// where builds the WHERE condition of a request, with its bound values, in a
// fixed order: the any-of filters in the registry's column order, then the
// ranges, then the sizes, then the words in the order typed. Only registry text
// is spliced into the SQL; every value is a placeholder.
func where(c *Collection, req listRequest) (string, []any) {
	var conds []string
	var args []any
	next := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	for _, f := range req.filters {
		conds = append(conds, f.sql(next(f.arg)))
	}
	for _, r := range req.ranges {
		conds = append(conds, r.sql(next))
	}
	if len(req.sizes) > 0 {
		conds = append(conds, strings.ReplaceAll(c.Sizes, "{s}", next(req.sizes)))
	}
	for _, w := range req.words {
		// The word is folded in SQL, once per statement: an uncorrelated scalar
		// subquery is evaluated once, where doi_fold($n) written inline would
		// be evaluated for every shoe.
		ph := "(SELECT doi_fold(" + next(w) + "))"
		conds = append(conds, "("+strings.ReplaceAll(c.Search, "{w}", ph)+")")
	}
	if len(conds) == 0 {
		return "TRUE", nil
	}
	return strings.Join(conds, " AND "), args
}

// orderBy is the order of a list: the chosen field, then the seed's order, so
// equal values keep a fixed order and a page never repeats or skips a record.
func orderBy(req listRequest) string {
	if req.sort == nil {
		return "t.seq"
	}
	return req.sort.sql() + ", t.seq ASC"
}

// listSQL is the one statement of a list: the matching count and the page, as
// one snapshot, so a record added between two statements cannot make total
// disagree with the page. The page comes back as a JSON array built by the
// database from the public columns, in the requested order.
func listSQL(c *Collection, req listRequest) (string, []any) {
	cond, args := where(c, req)
	if !req.dealer && c.Live != "" {
		// Whoever is not the team sees only what is on sale, whatever else they
		// asked for: a ?status=draft finds nothing.
		if cond == "TRUE" {
			cond = c.Live
		} else {
			cond = "(" + cond + ") AND (" + c.Live + ")"
		}
	}
	n := len(args)
	order := orderBy(req)
	if req.sort == nil && c.DefaultOrder != "" {
		order = c.DefaultOrder
	}
	sql := "SELECT (SELECT count(*) FROM " + c.From + " WHERE " + cond + ") AS total, " +
<<<<<<< refs/heads/feature/manage-shoes-lines-categories-reviews-and-orders-be
		"coalesce((SELECT json_agg(p) FROM (SELECT " + c.viewList(req.dealer) + " FROM " + c.From +
		" WHERE " + cond + " ORDER BY " + orderBy(req) + " LIMIT $" + strconv.Itoa(n+1) + " OFFSET $" + strconv.Itoa(n+2) +
=======
		"coalesce((SELECT json_agg(p) FROM (SELECT " + c.selectServing(req.served) + " FROM " + c.From +
		" WHERE " + cond + " ORDER BY " + order + " LIMIT $" + strconv.Itoa(n+1) + " OFFSET $" + strconv.Itoa(n+2) +
>>>>>>> refs/heads/main
		") p), '[]'::json)::text AS items"
	args = append(args, req.limit, (req.page-1)*req.limit)
	return sql, args
}

// recordSQL is the one statement of a record: its columns as a JSON object. A
// caller who is not the team gets the public columns, and only a record that is
// on sale: a draft answers as an id that does not exist.
func recordSQL(c *Collection, dealer bool) string {
	live := ""
	if !dealer && c.Live != "" {
		live = " AND (" + c.Live + ")"
	}
	return "SELECT row_to_json(p)::text FROM (SELECT " + c.viewList(dealer) + " FROM " + c.From + " WHERE t.id = $1" + live + ") p"
}
