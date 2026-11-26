package catalogue

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Querier is the slice of *pgxpool.Pool the package needs: its one statement
// returns one row, so one method is all a test has to fake.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// colourOrder is the closed set of main colours, in the order the shop lists
// them (ck_products_colour). A test checks it against the schema's options.
var colourOrder = []string{"black", "white", "pink", "purple", "brown", "yellow", "blue", "grey", "green"}

// filtersSQL reads every colour and every listed size of the live shoes, as two
// JSON arrays, in one statement. A size is listed whether or not it is sold
// out: a listed size is a size the shop sells. Drafts never contribute, to
// anyone: no session is read.
const filtersSQL = `SELECT
    coalesce((SELECT json_agg(c.colour ORDER BY array_position($1::text[], c.colour))
              FROM (SELECT DISTINCT t.colour FROM products t
                    WHERE t.status = 'live' AND t.colour <> '') c), '[]'::json)::text AS colours,
    coalesce((SELECT json_agg(s.size)
              FROM (SELECT DISTINCT z.size FROM product_sizes z JOIN products t ON t.id = z.product_id
                    WHERE t.status = 'live') s), '[]'::json)::text AS sizes`

// Filters is the shop's options: every colour and EU size a live shoe has.
type Filters struct {
	Colours []string `json:"colours"`
	Sizes   []string `json:"sizes"`
}

type store struct{ db Querier }

func (s store) filters(ctx context.Context) (Filters, error) {
	var colours, sizes string
	if err := s.db.QueryRow(ctx, filtersSQL, colourOrder).Scan(&colours, &sizes); err != nil {
		return Filters{}, err
	}
	out := Filters{Colours: []string{}, Sizes: []string{}}
	if err := json.Unmarshal([]byte(colours), &out.Colours); err != nil {
		return Filters{}, err
	}
	if err := json.Unmarshal([]byte(sizes), &out.Sizes); err != nil {
		return Filters{}, err
	}
	sortSizes(out.Sizes)
	return out, nil
}

// sizeValue reads an EU size as a number, so sizes sort the way a foot grows:
// "35 1/2" is 35.5, "36 2/3" is 36 and two thirds, "40" is 40. A size that does
// not read as a number says so.
func sizeValue(s string) (float64, bool) {
	whole, frac, hasFrac := strings.Cut(strings.TrimSpace(s), " ")
	v, err := strconv.ParseFloat(whole, 64)
	if err != nil {
		return 0, false
	}
	if !hasFrac {
		return v, true
	}
	num, den, ok := strings.Cut(strings.TrimSpace(frac), "/")
	if !ok {
		return 0, false
	}
	a, errA := strconv.ParseFloat(num, 64)
	b, errB := strconv.ParseFloat(den, 64)
	if errA != nil || errB != nil || b == 0 {
		return 0, false
	}
	return v + a/b, true
}

// sortSizes puts sizes in growing order; one that is not a number goes last, by
// text.
func sortSizes(sizes []string) {
	sort.SliceStable(sizes, func(i, j int) bool {
		vi, oki := sizeValue(sizes[i])
		vj, okj := sizeValue(sizes[j])
		switch {
		case oki && okj:
			if vi != vj {
				return vi < vj
			}
			return sizes[i] < sizes[j]
		case oki:
			return true
		case okj:
			return false
		}
		return sizes[i] < sizes[j]
	})
}
