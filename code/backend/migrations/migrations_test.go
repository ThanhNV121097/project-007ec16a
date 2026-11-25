package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var upName = regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.up\.sql$`)

// Every migration is a well-named pair with a version nobody else uses.
func TestMigrationFilesArePairs(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	versions := map[string]string{}
	for name := range names {
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			if !upName.MatchString(name) {
				t.Errorf("%s: name must be YYYYMMDDHHMMSS_description.up.sql", name)
				continue
			}
			if other, dup := versions[name[:14]]; dup {
				t.Errorf("%s and %s share the version %s", name, other, name[:14])
			}
			versions[name[:14]] = name
			if down := strings.TrimSuffix(name, ".up.sql") + ".down.sql"; !names[down] {
				t.Errorf("%s has no %s", name, down)
			}
		case strings.HasSuffix(name, ".down.sql"):
			if up := strings.TrimSuffix(name, ".down.sql") + ".up.sql"; !names[up] {
				t.Errorf("%s has no %s", name, up)
			}
		}
	}
}
