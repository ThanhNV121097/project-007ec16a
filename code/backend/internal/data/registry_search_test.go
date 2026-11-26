package data

import (
	"regexp"
	"strings"
	"testing"
)

// A column named like a range key would be swallowed by the filter rule before
// the range rule could read it.
func TestNoColumnEndsLikeARangeKey(t *testing.T) {
	for _, c := range registry {
		for _, col := range c.Columns {
			for _, suffix := range []string{"_from", "_to", "_ranges"} {
				if strings.HasSuffix(col.Name, suffix) {
					t.Errorf("%s.%s ends in %s", c.Name, col.Name, suffix)
				}
			}
		}
	}
}

// Only the shoes have an EU size condition; every collection can be searched.
func TestOnlyShoesHaveASizeCondition(t *testing.T) {
	for _, c := range registry {
		if (c.Sizes != "") != (c.Name == "products") {
			t.Errorf("%s: Sizes is %q", c.Name, c.Sizes)
		}
		if c.Search == "" {
			t.Errorf("%s cannot be searched", c.Name)
		}
	}
	sizes := byName["products"].Sizes
	if !strings.Contains(sizes, "{s}") || !strings.Contains(sizes, "product_sizes") || !strings.Contains(sizes, "pairs > 0") {
		t.Errorf("the size condition reads the pairs left of each size in product_sizes: %s", sizes)
	}
	if strings.Contains(sizes, "sizes_out") || strings.Contains(sizes, "t.sizes") {
		t.Errorf("the size condition names a column products no longer has: %s", sizes)
	}
}

// A record with no value comes last, so exactly the columns that can hold none
// say so: a NOT NULL column with a NULLS clause would stop an index serving it.
func TestNullableColumnsAreTheOnesThatCanHoldNothing(t *testing.T) {
	want := map[string][]string{
		"lines":    {"since"},
		"reviews":  {"rating"},
		"sizes":    {"cm"},
		"products": {"line", "category", "rating"},
	}
	for _, c := range registry {
		var got []string
		for _, col := range c.Columns {
			if col.Nullable {
				got = append(got, col.Name)
			}
		}
		if strings.Join(got, ",") != strings.Join(want[c.Name], ",") {
			t.Errorf("%s: nullable columns %v, want %v", c.Name, got, want[c.Name])
		}
	}
	if col, _ := column(byName["products"], "rating"); col.SortExpr != "NULLIF(t.rating, 0)" {
		t.Errorf("a shoe's rating sorts as %q", col.SortExpr)
	}
}

// A column that is not public is neither searched, filtered, ranged nor sorted:
// whatever a request asks, the statement never names it.
func TestNoRequestMakesTheSQLNameANonPublicColumn(t *testing.T) {
	for _, c := range registry {
		for _, col := range c.Columns {
			if col.Access == Public {
				continue
			}
			for _, key := range []string{col.Name, col.Name + "_from", col.Name + "_to", col.Name + "_ranges"} {
				if _, faults := parse(t, c.Name, key+"=1"); faults[key] == "" {
					t.Errorf("%s?%s was accepted", c.Name, key)
				}
			}
			if _, faults := parse(t, c.Name, "sort="+col.Name); faults["sort"] == "" {
				t.Errorf("%s?sort=%s was accepted", c.Name, col.Name)
			}
			if col.Select == "" {
				continue
			}
			name := regexp.MustCompile(`\b` + regexp.QuoteMeta(strings.TrimPrefix(col.Select, "t.")) + `\b`)
			if name.MatchString(c.Search) {
				t.Errorf("%s: the search names %s", c.Name, col.Name)
			}
		}
	}
}
