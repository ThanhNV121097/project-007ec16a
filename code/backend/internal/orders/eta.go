package orders

import (
	"errors"
	"regexp"
	"strconv"
	"time"
)

// ict is Vietnam time, a fixed offset: Vietnam has no daylight saving and a fixed
// zone needs no tzdata in the image.
var ict = time.FixedZone("ICT", 7*3600)

var digitRun = regexp.MustCompile(`[0-9]+`)

// largestNumber is the biggest whole number in a text such as "2–4" (4).
func largestNumber(s string) (int, error) {
	found := digitRun.FindAllString(s, -1)
	if len(found) == 0 {
		return 0, errors.New("standard_days holds no number")
	}
	most := 0
	for _, f := range found {
		n, err := strconv.Atoi(f)
		if err != nil {
			return 0, err
		}
		if n > most {
			most = n
		}
	}
	return most, nil
}

func weekend(d time.Time) bool { return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday }

// nextWeekday is the next Monday-to-Friday day after d.
func nextWeekday(d time.Time) time.Time {
	d = d.AddDate(0, 0, 1)
	for weekend(d) {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// expectedDate is the day an order placed at now is expected: the base day plus
// days working days, in Vietnam time. The base day is today, or the next working
// day when today is a Saturday or Sunday or now is at or after the dispatch
// cut-off ("15:00"). Vietnamese holidays are not skipped. The result is a date at
// midnight in Vietnam time.
func expectedDate(now time.Time, days int, cutoff string) (time.Time, error) {
	cut, err := time.Parse("15:04", cutoff)
	if err != nil {
		return time.Time{}, err
	}
	now = now.In(ict)
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ict)
	if weekend(base) || now.Hour()*60+now.Minute() >= cut.Hour()*60+cut.Minute() {
		base = nextWeekday(base)
	}
	eta := base
	for i := 0; i < days; i++ {
		eta = nextWeekday(eta)
	}
	return eta, nil
}
