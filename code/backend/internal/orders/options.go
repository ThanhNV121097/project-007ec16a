package orders

import (
	"context"
	"net/http"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// option is one way to pay the shop can complete, as Checkout step 03 shows it.
type option struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Note        string `json:"note"`
	Icon        string `json:"icon"`
	Online      bool   `json:"online"`
	IsAvailable bool   `json:"is_available"`
}

type optionList struct {
	Items []option `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}

func (s *service) available(id string) bool {
	// A way to pay with no provider registered is reported available: the key
	// check belongs to the story that registers the provider.
	if p := s.providers[id]; p != nil {
		return p.Available()
	}
	return true
}

// paymentOptions answers GET /v1/payment-options: the ways to pay the shop was
// told it can complete, in the order it listed them (vnpay, bank, momo), not the
// order of the rows. cod keeps its row in payment_methods and is never listed.
func (s *service) paymentOptions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	rows, err := s.db.Query(ctx, "SELECT id, name, note, icon, online FROM payment_methods WHERE id = ANY($1) ORDER BY array_position($1::text[], id)", s.offered)
	if err != nil {
		s.internal(w, r, "read payment options", err)
		return
	}
	defer rows.Close()
	items := []option{}
	for rows.Next() {
		var o option
		if err := rows.Scan(&o.ID, &o.Name, &o.Note, &o.Icon, &o.Online); err != nil {
			s.internal(w, r, "read payment options", err)
			return
		}
		o.IsAvailable = s.available(o.ID)
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		s.internal(w, r, "read payment options", err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, optionList{Items: items, Total: len(items), Page: 1, Limit: 100})
}
