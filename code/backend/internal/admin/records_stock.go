package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
	"github.com/jackc/pgx/v5"
)

// The pairs in each size of a shoe. A shoe's `pairs` is an object of size to
// whole number; its `sizes`, `sizes_low`, `sizes_out` and `stock` are derived
// from the counts by the view product_stock and are never written. Every write
// to a count goes through internal/stock.

const (
	recEnterPairs = "Enter the pairs in this size"
	recWholePairs = "Use a whole number, 0 or more"
	recManyPairs  = "Use a whole number up to 1,000,000"
	recBadSize    = "Use a size such as 38, 36 2/3 or 35.5."
	recTwice      = "This size is listed twice."
	recTaken      = "Pairs were just taken by an order. Check the number and save again."
	// recMaxPairs is the most pairs one size may hold. It keeps the sum of a
	// shoe's sizes (the view product_stock's integer) far from overflowing, which
	// would make every read that selects the shoe fail.
	recMaxPairs = 1_000_000
)

// stockSize is one size: as it is written and listed, its value as a number (the
// order sizes grow in), and its count of pairs.
type stockSize struct {
	size  string
	value float64
	pairs int
}

// stockPlan is what a body said about the counts of a shoe.
//
// pairs makes the listed sizes authoritative: a size left out is taken off the
// shoe with its count. seen is the guard of a change: the counts the form last
// read. A size whose typed count equals the one seen was not touched, and the
// shop keeps what it holds; a size whose count changed is written only if the
// shop still holds the one seen, and an order that moved it since is a 409 with
// nothing saved. A body with no seen counts is authoritative as it stands.
type stockPlan struct {
	hasPairs, hasSizes, hasSeen bool
	pairs                       []stockSize
	sizes                       []stockSize
	seen                        map[string]int
}

func (p stockPlan) touched() bool { return p.hasPairs || p.hasSizes }

// readStock judges the `pairs`, `sizes` and `seen_pairs` of a body. A fault is
// keyed `pairs.<size>`, `sizes` or `seen_pairs`.
func readStock(obj map[string]json.RawMessage, create bool, faults map[string]string) stockPlan {
	var p stockPlan
	if raw, ok := obj["pairs"]; ok {
		p.hasPairs = true
		p.pairs = readCounts(raw, "pairs", faults)
	}
	if raw, ok := obj["sizes"]; ok {
		p.hasSizes = true
		p.sizes = readSizeList(raw, faults)
		if p.hasPairs && faults["pairs"] == "" && faults["sizes"] == "" && !sameSizes(p.pairs, p.sizes) {
			faults["sizes"] = "The sizes must be the sizes that have a count."
		}
	}
	if raw, ok := obj["seen_pairs"]; ok {
		if create {
			faults["seen_pairs"] = "This field is not accepted when a shoe is added."
		} else {
			p.hasSeen = true
			p.seen = map[string]int{}
			for _, s := range readCounts(raw, "seen_pairs", faults) {
				p.seen[s.size] = s.pairs
			}
		}
	}
	return p
}

// faultSize is a size as a fault's name may show it: cut short when it is long.
func faultSize(size string) string {
	if r := []rune(size); len(r) > stock.MaxSizeRunes {
		return string(r[:stock.MaxSizeRunes])
	}
	return size
}

// readCounts reads an object of size to whole number. A key the shop cannot read
// as a size, a size written twice and a count that is not a whole number 0 or
// more are faults named `<field>.<size>`; a count left empty asks for itself.
func readCounts(raw json.RawMessage, field string, faults map[string]string) []stockSize {
	var obj map[string]json.RawMessage
	if isNull(raw) || json.Unmarshal(raw, &obj) != nil || obj == nil {
		faults[field] = "Send a size and a count for each size."
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []stockSize
	bySize, byValue := map[string]bool{}, map[string]bool{}
	for _, key := range keys {
		size, value, ok := stock.ParseSize(key)
		name := field + "." + faultSize(size)
		if !ok {
			faults[name] = recBadSize
			continue
		}
		label := fmt.Sprintf("%.2f", value)
		if bySize[size] || byValue[label] {
			faults[name] = recTwice
			continue
		}
		bySize[size], byValue[label] = true, true
		n, fault := readCount(obj[key], field == "pairs")
		if fault != "" {
			faults[name] = fault
			continue
		}
		out = append(out, stockSize{size: size, value: value, pairs: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].value < out[j].value || (out[i].value == out[j].value && out[i].size < out[j].size) })
	return out
}

// readCount reads one count. A count left empty (null or "") is asked for when
// the person is typing it.
func readCount(raw json.RawMessage, asking bool) (int, string) {
	var text string
	if asking && (isNull(raw) || (json.Unmarshal(raw, &text) == nil && strings.TrimSpace(text) == "")) {
		return 0, recEnterPairs
	}
	n, ok := wholeNumber(raw)
	if !ok || n < 0 {
		return 0, recWholePairs
	}
	if n > recMaxPairs {
		return 0, recManyPairs
	}
	return int(n), ""
}

// readSizeList reads `sizes`: a list of sizes as people write them.
func readSizeList(raw json.RawMessage, faults map[string]string) []stockSize {
	var list []string
	if isNull(raw) || json.Unmarshal(raw, &list) != nil {
		faults["sizes"] = "Send the sizes as a list."
		return nil
	}
	var out []stockSize
	seen := map[string]bool{}
	for _, item := range list {
		size, value, ok := stock.ParseSize(item)
		if !ok {
			faults["sizes"] = recBadSize
			return nil
		}
		if seen[size] {
			faults["sizes"] = recTwice
			return nil
		}
		seen[size] = true
		out = append(out, stockSize{size: size, value: value})
	}
	return out
}

func sameSizes(a, b []stockSize) bool {
	if len(a) != len(b) {
		return false
	}
	in := map[string]bool{}
	for _, s := range a {
		in[s.size] = true
	}
	for _, s := range b {
		if !in[s.size] {
			return false
		}
	}
	return true
}

// applyStock writes a shoe's counts under the transaction that saves the shoe.
// The shoe's counts are locked first, so an order placing pairs of it finishes
// before this reads them. A refusal (422 for a size listed without a count, 409
// for a count an order moved) is returned with nothing written; the caller rolls
// the transaction back.
func applyStock(ctx context.Context, tx pgx.Tx, id string, p stockPlan) (*recRefusal, error) {
	stored, err := stock.Pairs(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	final := map[string]stockSize{}
	if p.hasPairs {
		for _, s := range p.pairs {
			final[s.size] = s
		}
	} else {
		missing := map[string]string{}
		for _, s := range p.sizes {
			n, listed := stored[s.size]
			if !listed {
				missing["pairs."+faultSize(s.size)] = recEnterPairs
				continue
			}
			s.pairs = n
			final[s.size] = s
		}
		if len(missing) > 0 {
			return recInvalid(missing), nil
		}
	}

	conflicts := map[string]string{}
	var puts []stockSize
	for _, s := range sortedSizes(final) {
		current, listed := stored[s.size]
		seen, wasSeen := p.seen[s.size]
		switch {
		case !p.hasSeen:
			if !listed || current != s.pairs {
				puts = append(puts, s)
			}
		case listed && wasSeen && s.pairs == seen:
			// Not touched: the shop keeps what it holds, an order's pairs included.
		case listed && wasSeen && current != seen:
			conflicts["pairs."+faultSize(s.size)] = recTaken
		case listed && !wasSeen && current != s.pairs:
			conflicts["pairs."+faultSize(s.size)] = recTaken
		case !listed && wasSeen:
			conflicts["pairs."+faultSize(s.size)] = recTaken
		case !listed || (wasSeen && current == seen):
			puts = append(puts, s)
		}
	}
	var drop []string
	for size, current := range stored {
		if _, kept := final[size]; kept {
			continue
		}
		if seen, wasSeen := p.seen[size]; p.hasSeen && wasSeen && current != seen {
			conflicts["pairs."+faultSize(size)] = recTaken
			continue
		}
		drop = append(drop, size)
	}
	if len(conflicts) > 0 {
		return &recRefusal{status: 409, code: "conflict", message: recTaken, fields: conflicts}, nil
	}
	for _, s := range puts {
		if err := stock.Put(ctx, tx, id, s.size, s.value, s.pairs); err != nil {
			return nil, err
		}
	}
	sort.Strings(drop)
	return nil, stock.Remove(ctx, tx, id, drop)
}

func sortedSizes(m map[string]stockSize) []stockSize {
	out := make([]stockSize, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].value < out[j].value || (out[i].value == out[j].value && out[i].size < out[j].size) })
	return out
}
