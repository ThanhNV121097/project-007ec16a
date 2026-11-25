package data

import (
	"encoding/json"
	"os"
	"testing"
)

func column(c *Collection, name string) (Column, bool) {
	for _, col := range c.Columns {
		if col.Name == name {
			return col, true
		}
	}
	return Column{}, false
}

// The registry is a hand-written mirror of schema.json. This is the drift
// guard: it fails when a schema field has no column, a column has no schema
// field, or a field the schema marks private is served. It is also the tripwire
// for "Free delivery on every order" not having landed (free_delivery_from and
// the express fields would still be in the schema).
func TestRegistryMirrorsTheSchema(t *testing.T) {
	raw, err := os.ReadFile("../../../frontend/src/data/schema.json")
	if err != nil {
		t.Skipf("schema.json is not beside the backend (a backend-only checkout): %v", err)
	}
	var schema struct {
		Collections map[string]struct {
			Fields []struct {
				Key     string `json:"key"`
				Private bool   `json:"private"`
			} `json:"fields"`
		} `json:"collections"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Collections) != len(registry) {
		t.Errorf("schema has %d collections, the registry %d", len(schema.Collections), len(registry))
	}
	for name, sc := range schema.Collections {
		c, ok := byName[name]
		if !ok {
			t.Errorf("schema collection %q is not in the registry", name)
			continue
		}
		keys := map[string]bool{"id": true, "_source": true, "_invented": true}
		for _, f := range sc.Fields {
			keys[f.Key] = true
			col, ok := column(c, f.Key)
			if !ok {
				t.Errorf("%s.%s is in the schema and not in the registry", name, f.Key)
				continue
			}
			if f.Private && col.Access != Private {
				t.Errorf("%s.%s is private in the schema and %d in the registry", name, f.Key, col.Access)
			}
		}
		for _, col := range c.Columns {
			if !keys[col.Name] {
				t.Errorf("%s.%s is in the registry and not in the schema", name, col.Name)
			}
		}
	}
}

func TestBookkeepingAndCustomerDetailsAreNeverPublic(t *testing.T) {
	if len(registry) != 10 {
		t.Fatalf("the registry holds %d collections, want 10", len(registry))
	}
	for _, c := range registry {
		for _, name := range []string{"_source", "_invented"} {
			if col, ok := column(c, name); !ok || col.Access == Public {
				t.Errorf("%s.%s must exist and not be public", c.Name, name)
			}
		}
	}
	if col, _ := column(byName["products"], "price_listed"); col.Access == Public {
		t.Error("products.price_listed is public")
	}
	for _, name := range []string{"phone", "email", "address", "note"} {
		if col, ok := column(byName["orders"], name); !ok || col.Access != Private {
			t.Errorf("orders.%s must exist and be private", name)
		}
	}
	// private is per collection: note is an ordinary field of delivery and payments.
	for _, name := range []string{"delivery", "payments"} {
		if col, _ := column(byName[name], "note"); col.Access != Public {
			t.Errorf("%s.note must be public", name)
		}
	}
}

func TestPublicColumnCounts(t *testing.T) {
	want := map[string]int{
		"products": 29, "orders": 16, "lines": 5, "categories": 4, "reviews": 9,
		"shop": 6, "delivery": 5, "payments": 5, "faqs": 4, "sizes": 6,
	}
	for name, n := range want {
		if got := len(byName[name].public()); got != n {
			t.Errorf("%s serves %d columns, want %d", name, got, n)
		}
	}
}
