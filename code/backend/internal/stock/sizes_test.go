package stock

import "testing"

func TestParseSizeReadsASizeAsPeopleWriteIt(t *testing.T) {
	cases := []struct {
		in    string
		size  string
		value float64
		ok    bool
	}{
		{"38", "38", 38, true},
		{"  36   2/3 ", "36 2/3", 36.67, true},
		{"35 1/2", "35 1/2", 35.5, true},
		{"35.5", "35.5", 35.5, true},
		{"", "", 0, false},
		{"abc", "abc", 0, false},
		{"36 2/0", "36 2/0", 0, false},
		{"0", "0", 0, false},
		{"-3", "-3", 0, false},
		{"36 2 / 3", "36 2 / 3", 0, false},
		{"1234", "1234", 0, false},
	}
	for _, c := range cases {
		size, value, ok := ParseSize(c.in)
		if ok != c.ok || (ok && (size != c.size || value != c.value)) {
			t.Errorf("ParseSize(%q) = %q, %v, %v; want %q, %v, %v", c.in, size, value, ok, c.size, c.value, c.ok)
		}
	}
}

func TestParseSizeRefusesMoreThanTwentyCharacters(t *testing.T) {
	if _, _, ok := ParseSize("35 123/456 and more text here"); ok {
		t.Fatal("a size over 20 characters was accepted")
	}
}
