package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

const (
	// uploadMax is 5 MB read as 5 MiB: exactly this is accepted, one byte more is not.
	uploadMax = 5 << 20
	// uploadForm is the whole request: the file and the form's own framing.
	uploadForm = uploadMax + 64<<10

	tooLarge   = "That picture is too large (5 MB at most)."
	wrongKind  = "Choose a JPEG, PNG or WebP picture."
	formShape  = "Send the picture as a form with one part, file."
	cacheYear  = "public, max-age=31536000, immutable"
	uploadBase = "/api/v1/uploads/"
)

// extensions are the types a picture may be, by what its content says it is.
var extensions = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

var uploadName = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp)$`)

// upload answers POST /v1/uploads: one multipart part, file, a JPEG, PNG or WebP
// by content, at most 5 MiB. The file is stored under the shop's own name; the
// name it was sent with is never read.
func (h *recHandler) upload(w http.ResponseWriter, r *http.Request) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "multipart/form-data" {
		refuse(w, http.StatusBadRequest, "bad_request", formShape, nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, uploadForm)
	form, err := r.MultipartReader()
	if err != nil {
		refuseForm(w, err)
		return
	}
	part, err := form.NextPart()
	if err != nil {
		refuseForm(w, err)
		return
	}
	if part.FormName() != "file" {
		refuse(w, http.StatusBadRequest, "bad_request", formShape, nil)
		return
	}
	bytes, err := io.ReadAll(io.LimitReader(part, uploadMax+1))
	if err != nil {
		refuseForm(w, err)
		return
	}
	if len(bytes) > uploadMax {
		refuse(w, http.StatusRequestEntityTooLarge, "payload_too_large", tooLarge, nil)
		return
	}
	if extra, err := form.NextPart(); err == nil {
		_ = extra.Close()
		refuse(w, http.StatusBadRequest, "bad_request", formShape, nil)
		return
	} else if !errors.Is(err, io.EOF) {
		refuseForm(w, err)
		return
	}
	if len(bytes) == 0 {
		invalid(w, map[string]string{"file": "Choose a picture."})
		return
	}
	// The content decides, never the name or the type the part declares.
	kind := http.DetectContentType(bytes[:min(len(bytes), 512)])
	ext, ok := extensions[kind]
	if !ok {
		refuse(w, http.StatusUnsupportedMediaType, "unsupported_media_type", wrongKind, nil)
		return
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		h.fail(w, "uploads.name", err)
		return
	}
	name := hex.EncodeToString(raw[:]) + "." + ext
	var by any
	if u, signedIn := auth.FromContext(r.Context()); signedIn {
		by = u.ID
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	if _, err := h.db.Exec(ctx, `INSERT INTO uploads (name, content_type, size_bytes, data, uploaded_by) VALUES ($1, $2, $3, $4, $5::uuid)`,
		name, kind, len(bytes), bytes, by); err != nil {
		h.fail(w, "uploads.insert", err)
		return
	}
	url := uploadBase + name
	w.Header().Set("Location", url)
	reply(w, http.StatusCreated, map[string]string{"url": url})
}

// refuseForm answers a request that could not be read as the one-part form: 413
// when it is over the cap, 400 otherwise.
func refuseForm(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		refuse(w, http.StatusRequestEntityTooLarge, "payload_too_large", tooLarge, nil)
		return
	}
	refuse(w, http.StatusBadRequest, "bad_request", formShape, nil)
}

// picture answers GET /v1/uploads/{name} to everyone: the stored bytes, kept
// for ever by whoever holds them (the name is random and never reused).
func (h *recHandler) picture(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !uploadName.MatchString(name) {
		recNotFound(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	var kind string
	var bytes []byte
	err := h.db.QueryRow(ctx, `SELECT content_type, data FROM uploads WHERE name = $1`, name).Scan(&kind, &bytes)
	if errors.Is(err, pgx.ErrNoRows) {
		recNotFound(w)
		return
	}
	if err != nil {
		h.fail(w, "uploads.read", err)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", kind)
	hd.Set("Content-Length", strconv.Itoa(len(bytes)))
	hd.Set("Cache-Control", cacheYear)
	hd.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}
