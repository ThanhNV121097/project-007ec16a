package orders

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"slices"
	"strconv"
)

const (
	bodySentence      = "Send the order as one JSON object."
	undefinedSentence = "The shop does not take this field."
	textSentence      = "Send this as text."
	wholeSentence     = "Send this as a whole number."
	listSentence      = "Send the pairs as a list."
	pairSentence      = "Send each pair as an object."
	// maxFaults bounds how many faults of the body are named: a body of 64 KiB
	// cannot make the answer larger than the request.
	maxFaults = 50
)

// object reads raw as exactly one JSON object and leaves its values undecoded, so
// that each can be read as the type its field has and named when it is not.
func object(raw []byte) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var fields map[string]json.RawMessage
	if err := dec.Decode(&fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, errors.New("not an object")
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("data after the object")
	}
	return fields, nil
}

func text(dst *string, name string, raw json.RawMessage, faults fieldErrors) {
	if err := json.Unmarshal(raw, dst); err != nil {
		faults[name] = textSentence
	}
}

func whole(dst *int, name string, raw json.RawMessage, faults fieldErrors) {
	if err := json.Unmarshal(raw, dst); err != nil {
		faults[name] = wholeSentence
	}
}

// parseOrder reads a request body against the contract's fields and names every
// one that is wrong: a field the contract does not define, a value of the wrong
// type, a body that is not one object. Nothing is ignored but the four numbers the
// browser may send and the shop never trusts (price and unit_price of a pair,
// delivery_fee and total of the order). A non-empty result is a 400.
func parseOrder(raw []byte) (orderRequest, fieldErrors) {
	var req orderRequest
	fields, err := object(raw)
	if err != nil {
		return req, fieldErrors{"body": bodySentence}
	}
	texts := map[string]*string{
		"delivery": &req.Delivery, "payment": &req.Payment, "customer": &req.Customer,
		"phone": &req.Phone, "address": &req.Address, "city": &req.City, "note": &req.Note,
	}
	faults := fieldErrors{}
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		if len(faults) >= maxFaults {
			break
		}
		v := fields[name]
		switch dst := texts[name]; {
		case name == "items":
			req.Items = parseItems(v, faults)
		case dst != nil:
			text(dst, name, v, faults)
		case name == "delivery_fee":
			req.DeliveryFee = v
		case name == "total":
			req.Total = v
		default:
			faults[name] = undefinedSentence
		}
	}
	return req, faults
}

// parseItems reads the list of pairs. A fault is named items.<index> or
// items.<index>.<field>, the way validate names a pair. Only the first 51 pairs
// are read: validate refuses a list over 50 whatever the rest holds.
func parseItems(raw json.RawMessage, faults fieldErrors) []itemRequest {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		faults["items"] = listSentence
		return nil
	}
	if len(list) > maxPairs+1 {
		list = list[:maxPairs+1]
	}
	items := make([]itemRequest, len(list))
	for i, el := range list {
		name := "items." + strconv.Itoa(i)
		fields, err := object(el)
		if err != nil {
			faults[name] = pairSentence
			continue
		}
		it := &items[i]
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if len(faults) >= maxFaults {
				return items
			}
			v := fields[key]
			switch key {
			case "product":
				text(&it.Product, name+".product", v, faults)
			case "size":
				text(&it.Size, name+".size", v, faults)
			case "qty":
				whole(&it.Qty, name+".qty", v, faults)
			case "price":
				it.Price = v
			case "unit_price":
				it.UnitPrice = v
			default:
				faults[name+"."+key] = undefinedSentence
			}
		}
	}
	return items
}
