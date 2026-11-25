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

	badParameters = "Some parameters are not valid."
)

var (
	wholeNumber  = regexp.MustCompile(`^-?[0-9]{1,18}$`)
	decimalNum   = regexp.MustCompile(`^-?[0-9]{1,18}(\.[0-9]{1,6})?$`)
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

// filter is one exact-match condition: the SQL on the left of the comparison
// (from the registry) and the value to bind.
type filter struct {
	col Column
	arg any
}

// listRequest is a validated GET /v1/<collection>.
type listRequest struct {
	page    int64
	limit   int64
	filters []filter
}

// parseList reads the query string of a list request. It returns every fault
// at once, page and limit first; with any fault nothing is returned.
//
// Reserved and unknown parameters (q, sort, <field>_from, <field>_to, a name
// that is no field of the collection) are ignored, and so is a filter whose
// value is empty or holds a comma: "any of" belongs to the search story. A
// filter on a column that is not public is a fault, whatever its value: the
// answer to it would reveal what the column hides.
func parseList(c *Collection, q url.Values) (listRequest, map[string]string) {
	faults := map[string]string{}
	req := listRequest{page: 1, limit: defaultLimit}

	if v, ok := q["page"]; ok {
		n, err := strconv.ParseUint(v[0], 10, 64)
		if !digitsOnly.MatchString(v[0]) || err != nil || n < 1 || n > maxPage {
			faults["page"] = "page must be a whole number from 1."
		} else {
			req.page = int64(n)
		}
	}
	if v, ok := q["limit"]; ok {
		req.limit = parseLimit(v[0], faults)
	}

	for _, col := range c.Columns {
		vals, present := q[col.Name]
		if !present {
			continue
		}
		if col.Access != Public {
			faults[col.Name] = col.Name + " cannot be used as a filter."
			continue
		}
		value := vals[0]
		if value == "" || strings.Contains(value, ",") {
			continue
		}
		arg, fault := bindValue(col, value)
		if fault != "" {
			faults[col.Name] = fault
			continue
		}
		req.filters = append(req.filters, filter{col: col, arg: arg})
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

// bindValue turns a filter value into what is bound to its placeholder, or the
// sentence that says why it cannot be read as the field's kind. A value
// PostgreSQL would refuse is refused here, before any statement is sent.
func bindValue(col Column, v string) (any, string) {
	if !readableText(v) {
		return nil, col.Name + " must be plain text."
	}
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

// where builds the WHERE condition of a request, with its bound values. Only
// registry text is spliced into the SQL; every value is a placeholder.
func where(filters []filter) (string, []any) {
	if len(filters) == 0 {
		return "TRUE", nil
	}
	conds := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	for i, f := range filters {
		placeholder := "$" + strconv.Itoa(i+1) + f.col.Kind.cast()
		if f.col.Kind == List {
			conds = append(conds, placeholder+" = ANY("+f.col.filterExpr()+")")
		} else {
			conds = append(conds, f.col.filterExpr()+" = "+placeholder)
		}
		args = append(args, f.arg)
	}
	return strings.Join(conds, " AND "), args
}

// listSQL is the one statement of a list: the matching count and the page, as
// one snapshot, so a record added between two statements cannot make total
// disagree with the page. The page comes back as a JSON array built by the
// database from the public columns, in seq order.
func listSQL(c *Collection, req listRequest) (string, []any) {
	cond, args := where(req.filters)
	n := len(args)
	sql := "SELECT (SELECT count(*) FROM " + c.From + " WHERE " + cond + ") AS total, " +
		"coalesce((SELECT json_agg(p) FROM (SELECT " + c.selectList() + " FROM " + c.From +
		" WHERE " + cond + " ORDER BY t.seq LIMIT $" + strconv.Itoa(n+1) + " OFFSET $" + strconv.Itoa(n+2) +
		") p), '[]'::json)::text AS items"
	args = append(args, req.limit, (req.page-1)*req.limit)
	return sql, args
}

// recordSQL is the one statement of a record: its public columns as a JSON
// object.
func recordSQL(c *Collection) string {
	return "SELECT row_to_json(p)::text FROM (SELECT " + c.selectList() + " FROM " + c.From + " WHERE t.id = $1) p"
}
