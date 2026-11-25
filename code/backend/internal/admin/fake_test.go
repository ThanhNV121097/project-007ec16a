package admin

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// The ids of the accounts a fakeStore starts with: canonical lower-case uuids.
const (
	adminID    = "00000000-0000-4000-8000-000000000001"
	staffID    = "00000000-0000-4000-8000-000000000002"
	offID      = "00000000-0000-4000-8000-000000000003"
	customerID = "00000000-0000-4000-8000-000000000004"
	unknownID  = "00000000-0000-4000-8000-0000000000ff"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeStore answers from memory what pgStore answers from PostgreSQL, and
// records what it was asked to write.
type fakeStore struct {
	mu     sync.Mutex
	users  []auth.User
	failed error // when set, every call fails with it
	next   int
	writes []string
	hashes map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users: []auth.User{
			{ID: adminID, Email: "dealer@example.com", Name: "", Role: auth.Admin, Active: true},
			{ID: staffID, Email: "minh@example.com", Name: "Minh Nguyen", Role: auth.Staff, Active: true},
			{ID: offID, Email: "hoa@example.com", Name: "Hoa Pham", Role: auth.Staff, Active: false},
			{ID: customerID, Email: "customer@example.com", Name: "Customer", Role: auth.Customer, Active: true},
		},
		hashes: map[string]string{},
	}
}

func (f *fakeStore) find(id string) (int, bool) {
	for i, u := range f.users {
		if u.ID == id && u.Role != auth.Customer {
			return i, true
		}
	}
	return 0, false
}

func (f *fakeStore) list(_ context.Context, limit, offset int64) ([]auth.User, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return nil, 0, f.failed
	}
	var team []auth.User
	for _, u := range f.users {
		if u.Role != auth.Customer {
			team = append(team, u)
		}
	}
	total := int64(len(team))
	lo := min(offset, total)
	hi := min(offset+limit, total)
	return team[lo:hi], total, nil
}

func (f *fakeStore) lookup(_ context.Context, id string) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return auth.User{}, f.failed
	}
	i, ok := f.find(id)
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return f.users[i], nil
}

func (f *fakeStore) emailTaken(_ context.Context, email string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return false, f.failed
	}
	for _, u := range f.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) create(_ context.Context, name, email, hash string) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return auth.User{}, f.failed
	}
	for _, u := range f.users {
		if u.Email == email {
			return auth.User{}, auth.ErrEmailTaken
		}
	}
	f.next++
	u := auth.User{ID: fmt.Sprintf("00000000-0000-4000-8000-%012x", 0x1000+f.next), Email: email, Name: name, Role: auth.Staff, Active: true}
	f.users = append(f.users, u)
	f.hashes[u.ID] = hash
	f.writes = append(f.writes, "create "+email)
	return u, nil
}

func (f *fakeStore) update(_ context.Context, id string, c change) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return auth.User{}, f.failed
	}
	i, ok := f.find(id)
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	if f.users[i].Role == auth.Admin {
		return auth.User{}, errAdminAccount
	}
	if c.name != nil {
		f.users[i].Name = *c.name
		f.writes = append(f.writes, "name "+*c.name)
	}
	if c.active != nil {
		f.users[i].Active = *c.active
		f.writes = append(f.writes, fmt.Sprintf("active %v", *c.active))
	}
	if c.hash != nil {
		f.hashes[id] = *c.hash
		f.writes = append(f.writes, "password")
	}
	return f.users[i], nil
}

func (f *fakeStore) remove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return f.failed
	}
	i, ok := f.find(id)
	if !ok || f.users[i].Role != auth.Staff {
		return auth.ErrNotFound
	}
	f.users = append(f.users[:i], f.users[i+1:]...)
	f.writes = append(f.writes, "remove "+id)
	return nil
}

// harness serves the four routes over a fakeStore, with whoever is "signed in"
// put in the context as Resolve would.
type harness struct {
	store *fakeStore
	mux   *http.ServeMux
}

func newHarness() *harness {
	h := &harness{store: newFakeStore(), mux: http.NewServeMux()}
	(&staffHandler{store: h.store, log: quiet()}).mount(h.mux)
	h.mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	return h
}

// do sends a request as role ("" is nobody). A body is sent as JSON unless
// contentType says otherwise.
func (h *harness) do(role auth.Role, method, path, body string, contentType ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" || method == http.MethodPost || method == http.MethodPut {
		ct := "application/json"
		if len(contentType) > 0 {
			ct = contentType[0]
		}
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
	}
	if role != "" {
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "00000000-0000-4000-8000-0000000000aa", Email: "who@example.com", Role: role, Active: true}))
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}
