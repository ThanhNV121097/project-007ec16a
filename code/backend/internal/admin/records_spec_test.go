package admin

import (
	"encoding/json"
	"strings"
	"testing"
)

func recBody(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func TestAShoeNeedsANameAndAPriceAndEveryFaultIsNamed(t *testing.T) {
	in := recSpecByName["products"].read(recBody(t, `{"price":-1}`), true)
	if in.faults["name"] != "Name is needed" || in.faults["price"] != "Price cannot be below 0" {
		t.Fatalf("faults %v", in.faults)
	}
}

func TestPriceIsAWholeNumberZeroOrMore(t *testing.T) {
	s := recSpecByName["products"]
	for _, c := range []struct {
		price string
		ok    bool
	}{{`0`, true}, {`3290000`, true}, {`-1`, false}, {`2.5`, false}, {`"abc"`, false}, {`null`, false}, {`1e3`, false}} {
		in := s.read(recBody(t, `{"name":"A","price":`+c.price+`}`), true)
		if _, bad := in.faults["price"]; bad == c.ok {
			t.Errorf("price %s: faults %v", c.price, in.faults)
		}
	}
}

func TestAFieldTheCollectionDoesNotHaveOrTheShopWorksOutIsNamed(t *testing.T) {
	in := recSpecByName["products"].read(recBody(t, `{"name":"A","price":1,"colour_code":"x","rating":5,"review_count":1,"stock":3,"sizes_low":[],"sizes_out":[],"id":"mine"}`), true)
	for _, key := range []string{"colour_code", "rating", "review_count", "stock", "sizes_low", "sizes_out", "id"} {
		if in.faults[key] == "" {
			t.Errorf("%s was not refused: %v", key, in.faults)
		}
	}
	// A body's id on a change is read and ignored.
	if in := recSpecByName["products"].read(recBody(t, `{"id":"other","price":5}`), false); len(in.faults) != 0 || len(in.values) != 1 {
		t.Errorf("a change carrying an id: %v %v", in.faults, in.values)
	}
}

func TestOnlyTheFieldsSentAreJudgedOnAChange(t *testing.T) {
	in := recSpecByName["products"].read(recBody(t, `{"price":2600000}`), false)
	if len(in.faults) != 0 || len(in.values) != 1 || in.values[0].field.name != "price" || in.stock.touched() {
		t.Fatalf("%v %v", in.faults, in.values)
	}
	if in := recSpecByName["products"].read(recBody(t, `{"name":null}`), false); in.faults["name"] == "" {
		t.Error("a required field sent as null was accepted")
	}
}

func TestClosedSetsAndDates(t *testing.T) {
	in := recSpecByName["products"].read(recBody(t, `{"status":"archived","colour":"teal","released":"2026-13-40"}`), false)
	for _, key := range []string{"status", "colour", "released"} {
		if in.faults[key] == "" {
			t.Errorf("%s was accepted: %v", key, in.faults)
		}
	}
	if in := recSpecByName["products"].read(recBody(t, `{"status":"live","colour":"","released":"2026-11-26","line":""}`), false); len(in.faults) != 0 {
		t.Errorf("valid values refused: %v", in.faults)
	}
}

func TestAReviewsStarsAreOneToFiveOrNothing(t *testing.T) {
	s := recSpecByName["reviews"]
	for _, c := range []struct {
		stars string
		ok    bool
	}{{`1`, true}, {`5`, true}, {`null`, true}, {`6`, false}, {`0`, false}, {`2.5`, false}} {
		in := s.read(recBody(t, `{"product":"p","name":"A","rating":`+c.stars+`}`), true)
		if _, bad := in.faults["rating"]; bad == c.ok {
			t.Errorf("stars %s: %v", c.stars, in.faults)
		}
	}
	in := s.read(recBody(t, `{}`), true)
	if in.faults["product"] == "" || in.faults["name"] == "" {
		t.Errorf("a review without a shoe or a name: %v", in.faults)
	}
}

func TestLimitsCountCharactersNotBytes(t *testing.T) {
	s := recSpecByName["lines"]
	name := func(n int, r string) map[string]json.RawMessage {
		return recBody(t, `{"name":"`+strings.Repeat(r, n)+`"}`)
	}
	if in := s.read(name(200, "ạ"), true); len(in.faults) != 0 {
		t.Errorf("200 characters refused: %v", in.faults)
	}
	if in := s.read(name(201, "ạ"), true); !strings.Contains(in.faults["name"], "200") {
		t.Errorf("201 characters: %v", in.faults)
	}
	longText := func(n int) map[string]json.RawMessage {
		return recBody(t, `{"name":"A","blurb":"`+strings.Repeat("x", n)+`"}`)
	}
	if in := s.read(longText(5000), true); len(in.faults) != 0 {
		t.Errorf("5,000 characters refused: %v", in.faults)
	}
	if in := s.read(longText(5001), true); !strings.Contains(in.faults["blurb"], "5,000") {
		t.Errorf("5,001 characters: %v", in.faults)
	}
}

func TestAPictureIsAnAddressNeverTheFileItself(t *testing.T) {
	in := recSpecByName["categories"].read(recBody(t, `{"name":"A","image":"data:image/png;base64,AAAA"}`), true)
	if in.faults["image"] == "" {
		t.Fatal("a data: picture was accepted")
	}
}

func TestSlugs(t *testing.T) {
	for in, want := range map[string]string{
		"samba og new colour": "samba-og-new-colour",
		"  --Gazelle!! ":      "gazelle",
		"":                    "item",
		"???":                 "item",
	} {
		if got := slugOf(in); got != want {
			t.Errorf("slugOf(%q) = %q, want %q", in, got, want)
		}
	}
	if got := slugOf(strings.Repeat("a", 120)); len(got) != 80 {
		t.Errorf("a long name made %d characters", len(got))
	}
}

func TestThePairsOfAShoe(t *testing.T) {
	s := recSpecByName["products"]
	in := s.read(recBody(t, `{"name":"A","price":1,"pairs":{"38":4,"39":2,"40":0}}`), true)
	if len(in.faults) != 0 || !in.stock.touched() || len(in.stock.pairs) != 3 || in.stock.pairs[2].size != "40" || in.stock.pairs[2].pairs != 0 {
		t.Fatalf("%v %+v", in.faults, in.stock)
	}
	bad := s.read(recBody(t, `{"pairs":{"38":-1,"39":2.5,"40":"abc","41":null,"x":1,"36 2/3":1,"36  2/3":1}}`), false)
	for _, key := range []string{"pairs.38", "pairs.39", "pairs.40", "pairs.41", "pairs.x", "pairs.36 2/3"} {
		if bad.faults[key] == "" {
			t.Errorf("%s was accepted: %v", key, bad.faults)
		}
	}
	if bad.faults["pairs.41"] != recEnterPairs {
		t.Errorf("an empty count: %q", bad.faults["pairs.41"])
	}
}

func TestSizesMustAgreeWithPairsAndSeenPairsIsForAChangeOnly(t *testing.T) {
	s := recSpecByName["products"]
	if in := s.read(recBody(t, `{"sizes":["38","39"],"pairs":{"38":1}}`), false); in.faults["sizes"] == "" {
		t.Errorf("sizes that disagree with pairs: %v", in.faults)
	}
	if in := s.read(recBody(t, `{"sizes":["38","38"]}`), false); in.faults["sizes"] == "" {
		t.Errorf("a size listed twice: %v", in.faults)
	}
	if in := s.read(recBody(t, `{"name":"A","price":1,"pairs":{"38":1},"seen_pairs":{"38":1}}`), true); in.faults["seen_pairs"] == "" {
		t.Errorf("seen_pairs on a new shoe: %v", in.faults)
	}
	in := s.read(recBody(t, `{"pairs":{"38":6},"seen_pairs":{"38":4}}`), false)
	if len(in.faults) != 0 || !in.stock.hasSeen || in.stock.seen["38"] != 4 {
		t.Errorf("a guarded change: %v %+v", in.faults, in.stock)
	}
	if in := s.read(recBody(t, `{"price":5}`), false); in.stock.touched() || in.stock.hasSeen {
		t.Error("a change of the price alone carries counts")
	}
}
