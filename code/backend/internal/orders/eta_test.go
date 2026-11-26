package orders

import (
	"testing"
	"time"
)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, ict)
}

// SC-17: the expected date, in Vietnam time, skipping Saturdays and Sundays.
// 2026-11-23 is a Monday.
func TestExpectedDate(t *testing.T) {
	for _, c := range []struct {
		name string
		now  time.Time
		want string
	}{
		{"Monday 10:00", at(2026, 11, 23, 10, 0), "2026-11-27"},
		{"Monday 14:59", at(2026, 11, 23, 14, 59), "2026-11-27"},
		{"Monday 16:00, after the cut-off", at(2026, 11, 23, 16, 0), "2026-11-30"},
		{"Friday 15:00, at the cut-off", at(2026, 11, 27, 15, 0), "2026-12-04"},
		{"Friday 09:00", at(2026, 11, 27, 9, 0), "2026-12-03"},
		{"Saturday", at(2026, 11, 28, 11, 0), "2026-12-04"},
		{"Sunday evening", at(2026, 11, 29, 20, 0), "2026-12-04"},
	} {
		got, err := expectedDate(c.now, 4, "15:00")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Format("2006-01-02") != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got.Format("2006-01-02"), c.want)
		}
	}
}

// The day is the one in Vietnam, whatever zone the clock is in: 20:00 UTC on a
// Monday is already Tuesday 03:00 in Vietnam.
func TestExpectedDateReadsTheClockInVietnamTime(t *testing.T) {
	now := time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	got, err := expectedDate(now, 4, "15:00")
	if err != nil || got.Format("2006-01-02") != "2026-11-30" {
		t.Errorf("got %v %v, want 2026-11-30", got, err)
	}
}

func TestExpectedDateRefusesACutoffThatIsNotAClock(t *testing.T) {
	if _, err := expectedDate(at(2026, 11, 23, 10, 0), 4, "after lunch"); err == nil {
		t.Error("a cut-off that is not HH:MM was accepted")
	}
}

func TestLargestNumber(t *testing.T) {
	for in, want := range map[string]int{"2–4": 4, "2-4 working days": 4, "3": 3, "10 to 12": 12} {
		if got, err := largestNumber(in); err != nil || got != want {
			t.Errorf("%q: got %d %v, want %d", in, got, err, want)
		}
	}
	if _, err := largestNumber("soon"); err == nil {
		t.Error("a text with no number was accepted")
	}
}
