package payments

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// The text the design writes for BIN 970436, account 0071001234567, 3.140.000 VND and
// DOI-24818, up to the checksum.
const designPayload = "00020101021238570010A00000072701270006970436011300710012345670208QRIBFTTA5303704540731400005802VN62130809DOI-248186304"

// walk splits a string of EMVCo fields into tag -> value, and fails on a length that lies.
func walk(t *testing.T, s string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for len(s) > 0 {
		if len(s) < 4 {
			t.Fatalf("a truncated field: %q", s)
		}
		n, err := strconv.Atoi(s[2:4])
		if err != nil || len(s) < 4+n {
			t.Fatalf("a field whose length lies: %q", s)
		}
		out[s[:2]] = s[4 : 4+n]
		s = s[4+n:]
	}
	return out
}

// The algorithm's published check value.
func TestTheChecksumOfTheCheckStringIs29B1(t *testing.T) {
	if got := crc16("123456789"); got != 0x29B1 {
		t.Errorf("crc16 = %04X, want 29B1", got)
	}
}

// SC-12: the payload for the design's example is the design's text, its checksum
// validates, and a parse of it gives back the account, the amount and the note.
func TestThePayloadIsTheDesignsAndParsesBackToItsParts(t *testing.T) {
	for _, c := range []struct {
		amount int64
		note   string
	}{{3140000, "DOI-24818"}, {450000, "DOI-100"}, {9999999999999, "DOI-999999999999999"[:19]}} {
		p, err := vietQR("970436", "0071001234567", c.amount, c.note)
		if err != nil {
			t.Fatal(err)
		}
		if c.amount == 3140000 && (!strings.HasPrefix(p, designPayload) || len(p) != len(designPayload)+4) {
			t.Errorf("the payload is %s", p)
		}
		if got, want := p[len(p)-4:], fmt.Sprintf("%04X", crc16(p[:len(p)-4])); got != want {
			t.Errorf("the checksum %s does not validate (%s)", got, want)
		}
		f := walk(t, p)
		merchant := walk(t, f["38"])
		account := walk(t, merchant["01"])
		if f["00"] != "01" || f["01"] != "12" || f["53"] != "704" || f["58"] != "VN" || f["63"] != p[len(p)-4:] ||
			f["54"] != strconv.FormatInt(c.amount, 10) || walk(t, f["62"])["08"] != c.note ||
			merchant["00"] != "A000000727" || merchant["02"] != "QRIBFTTA" ||
			account["00"] != "970436" || account["01"] != "0071001234567" {
			t.Errorf("the parts: %v %v %v", f, merchant, account)
		}
	}
}

// What the settings never produce and the page must never ask for never reaches the payload.
func TestThePayloadBuilderRefusesWhatItShouldNeverBeGiven(t *testing.T) {
	for name, c := range map[string]struct {
		bin, account string
		amount       int64
		note         string
	}{
		"a BIN of five digits":         {"97043", "0071001234567", 1000, "DOI-1"},
		"a BIN with a letter":          {"97043x", "0071001234567", 1000, "DOI-1"},
		"an account with a space":      {"970436", "0071 0012", 1000, "DOI-1"},
		"an empty account":             {"970436", "", 1000, "DOI-1"},
		"an account of 20 characters":  {"970436", strings.Repeat("1", 20), 1000, "DOI-1"},
		"no amount":                    {"970436", "0071001234567", 0, "DOI-1"},
		"an amount of 14 digits":       {"970436", "0071001234567", 10_000_000_000_000, "DOI-1"},
		"a note over 25 characters":    {"970436", "0071001234567", 1000, strings.Repeat("A", 26)},
		"an empty note":                {"970436", "0071001234567", 1000, ""},
		"a note with a newline":        {"970436", "0071001234567", 1000, "DOI\n1"},
	} {
		if p, err := vietQR(c.bin, c.account, c.amount, c.note); err == nil {
			t.Errorf("%s: built %s", name, p)
		}
	}
}

// The picture is square, holds only dark and light modules and has no quiet zone of its own.
func TestTheQRCodeIsSquareRowsOfZeroAndOne(t *testing.T) {
	p, err := vietQR("970436", "0071001234567", 3140000, "DOI-24818")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := qrRows(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 21 || len(rows) > 77 {
		t.Fatalf("%d rows", len(rows))
	}
	for i, row := range rows {
		if len(row) != len(rows) || strings.Trim(row, "01") != "" {
			t.Fatalf("row %d: %q", i, row)
		}
	}
	// The finder pattern's corner is dark: the first module of the first row, with no margin before it.
	if rows[0][0] != '1' || rows[0][len(rows)-1] != '1' || rows[len(rows)-1][0] != '1' {
		t.Errorf("the finder corners: %c %c %c", rows[0][0], rows[0][len(rows)-1], rows[len(rows)-1][0])
	}
	again, _ := qrRows(p)
	if strings.Join(again, "") != strings.Join(rows, "") {
		t.Error("the same payload drew two pictures")
	}
}
