package admin

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// T1: the pure rules. Lengths are counted in characters, not bytes.

func TestNameRules(t *testing.T) {
	for _, c := range []struct {
		name, raw, want, fault string
	}{
		{"empty", "", "", "A name is needed"},
		{"only spaces", "   ", "", "A name is needed"},
		{"one character", "A", "A", ""},
		{"100 characters", strings.Repeat("a", 100), strings.Repeat("a", 100), ""},
		{"101 characters", strings.Repeat("a", 101), "", "The name can have at most 100 characters."},
		{"100 characters of 3 bytes", strings.Repeat("ế", 100), strings.Repeat("ế", 100), ""},
		{"trimmed before it is counted", " " + strings.Repeat("a", 100) + " ", strings.Repeat("a", 100), ""},
		{"stored trimmed", "  Lan Tran ", "Lan Tran", ""},
		{"NUL", "La\x00n", "", "The name cannot contain control characters."},
		{"a tab inside", "Lan\tTran", "", "The name cannot contain control characters."},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, fault := judgeName(c.raw)
			if got != c.want || fault != c.fault {
				t.Errorf("judgeName(%q) = %q, %q; want %q, %q", c.raw, got, fault, c.want, c.fault)
			}
		})
	}
}

func TestEmailRules(t *testing.T) {
	const looks = "That does not look like an email address."
	long := func(n int) string { return strings.Repeat("a", n-len("@e.com")) + "@e.com" }
	for _, c := range []struct {
		name, raw, want, fault string
	}{
		{"lower-cased", "Lan@Example.COM", "lan@example.com", ""},
		{"trimmed", "  lan@example.com ", "lan@example.com", ""},
		{"200 characters", long(200), long(200), ""},
		{"201 characters", long(201), "", looks},
		{"empty", "", "", "An email address is needed"},
		{"only spaces", "  ", "", "An email address is needed"},
		{"a non-ASCII letter", "dôi@example.com", "", looks},
		{"no @", "lan.example.com", "", looks},
		{"two @", "a@b@example.com", "", looks},
		{"no dot in the domain", "lan@example", "", looks},
		{"a space inside", "la n@example.com", "", looks},
		{"NUL", "lan\x00@example.com", "", looks},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, fault := judgeEmail(c.raw)
			if got != c.want || fault != c.fault {
				t.Errorf("judgeEmail(%q) = %q, %q; want %q, %q", c.raw, got, fault, c.want, c.fault)
			}
		})
	}
}

func TestPasswordRules(t *testing.T) {
	const short, long = "The password needs at least 10 characters.", "The password can have at most 100 characters."
	for _, c := range []struct {
		name, raw, fault string
	}{
		{"empty", "", short},
		{"9 characters", strings.Repeat("a", 9), short},
		{"10 characters", strings.Repeat("a", 10), ""},
		{"100 characters", strings.Repeat("a", 100), ""},
		{"101 characters", strings.Repeat("a", 101), long},
		{"10 Vietnamese characters (30 bytes)", strings.Repeat("ế", 10), ""},
		{"9 Vietnamese characters (27 bytes)", strings.Repeat("ế", 9), short},
		{"101 Vietnamese characters", strings.Repeat("ế", 101), long},
		{"spaces are kept and counted", "         a", ""},
		{"spaces are never trimmed away", "  abcdefg  ", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if fault := judgePassword(c.raw); fault != c.fault {
				t.Errorf("judgePassword = %q; want %q", fault, c.fault)
			}
		})
	}
}

func TestCreateNamesEveryFieldAtFault(t *testing.T) {
	_, _, faults := judgeCreate(input{Name: " ", Email: "nope", Password: "short"})
	for _, field := range []string{"name", "email", "password"} {
		if faults[field] == "" {
			t.Errorf("%s was not named: %v", field, faults)
		}
	}
	name, email, faults := judgeCreate(input{Name: " Lan Tran ", Email: "Lan@Example.com", Password: strings.Repeat("p", 12)})
	if len(faults) != 0 || name != "Lan Tran" || email != "lan@example.com" {
		t.Errorf("a good body: %q %q %v", name, email, faults)
	}
}

func TestUpdateJudgesOnlyWhatWasSentAndAlwaysRefusesRole(t *testing.T) {
	in := input{Has: map[string]bool{}}
	if faults := judgeUpdate(&in); len(faults) != 0 {
		t.Errorf("nothing sent: %v", faults)
	}
	in = input{Name: " Lan ", Has: map[string]bool{"name": true}}
	if faults := judgeUpdate(&in); len(faults) != 0 || in.Name != "Lan" {
		t.Errorf("a name: %v %q", faults, in.Name)
	}
	in = input{Name: " ", Password: "x", Has: map[string]bool{"name": true, "password": true, "role": true}}
	faults := judgeUpdate(&in)
	if faults["name"] == "" || faults["password"] == "" || faults["role"] != "The role cannot be changed." {
		t.Errorf("faults: %v", faults)
	}
}

func TestReadInputRefusesWhatTheContractDoesNotDefine(t *testing.T) {
	read := func(body string, spec map[string]kind) (input, map[string]string, bool) {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &obj); err != nil {
			t.Fatal(err)
		}
		return readInput(obj, spec)
	}
	for _, c := range []struct {
		name, body string
		spec       map[string]kind
		field      string
		want       string
	}{
		{"role on a POST", `{"name":"a","email":"a@b.co","password":"x","role":"admin"}`, createFields, "role", notAccepted},
		{"email on a PUT", `{"email":"a@b.co"}`, updateFields, "email", notAccepted},
		{"a number for a name", `{"name":5}`, updateFields, "name", notText},
		{"null for a name", `{"name":null}`, updateFields, "name", notText},
		{"a list for a password", `{"password":["x"]}`, updateFields, "password", notText},
		{"an object for an email", `{"name":"a","email":{},"password":"x"}`, createFields, "email", notText},
		{"a string for is_active", `{"is_active":"false"}`, updateFields, "is_active", notFlag},
		{"null for is_active", `{"is_active":null}`, updateFields, "is_active", notFlag},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, faults, bad := read(c.body, c.spec)
			if !bad || faults[c.field] != c.want {
				t.Errorf("bad=%v faults=%v; want %s: %q", bad, faults, c.field, c.want)
			}
		})
	}
	in, faults, bad := read(`{"name":"Lan","is_active":false,"password":"p","role":"admin","id":"x"}`, updateFields)
	if bad || len(faults) != 0 || !in.Has["role"] || !in.Has["id"] || in.Active || in.Name != "Lan" {
		t.Errorf("a PUT body the contract defines: %+v %v %v", in, faults, bad)
	}
	if _, faults, bad = read(`{"a b":1}`, updateFields); !bad || len(faults) != 0 {
		t.Errorf("a key that is not plain is refused without being echoed: %v %v", faults, bad)
	}
}

func TestPaging(t *testing.T) {
	for _, c := range []struct {
		query         string
		page, limit   int64
		faultyOn      string
		noFaultsAtAll bool
	}{
		{"", 1, 12, "", true},
		{"limit=100", 1, 100, "", true},
		{"limit=1000", 1, 100, "", true},
		{"limit=99999999999999999999999", 1, 100, "", true},
		{"page=2&limit=5", 2, 5, "", true},
		{"colour=red&page=3", 3, 12, "", true},
		{"page=0", 1, 12, "page", false},
		{"page=-1", 1, 12, "page", false},
		{"page=abc", 1, 12, "page", false},
		{"page=", 1, 12, "page", false},
		{"page=2147483648", 1, 12, "page", false},
		{"limit=0", 1, 12, "limit", false},
		{"limit=1.5", 1, 12, "limit", false},
		{"limit=", 1, 12, "limit", false},
	} {
		t.Run(c.query, func(t *testing.T) {
			q, _ := url.ParseQuery(c.query)
			page, limit, faults := readPaging(q)
			if c.noFaultsAtAll && len(faults) != 0 || !c.noFaultsAtAll && faults[c.faultyOn] == "" {
				t.Fatalf("faults = %v", faults)
			}
			if c.noFaultsAtAll && (page != c.page || limit != c.limit) {
				t.Errorf("page=%d limit=%d; want %d %d", page, limit, c.page, c.limit)
			}
		})
	}
}
