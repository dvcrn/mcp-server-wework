package app

import (
	"testing"
	"time"

	"github.com/dvcrn/wework-cli/pkg/wework"
)

func tokyoBooking(t *testing.T) *wework.Booking {
	t.Helper()

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	booking := &wework.Booking{
		BookingID: "1000001",
		TimeZone:  "Asia/Tokyo",
		SpaceID:   "00000000-0000-4000-8000-000000000001",
		SpaceName: "Daily Desk",
		Location: &wework.BookingLocation{
			UUID:     "00000000-0000-4000-8000-000000000002",
			Name:     "Shibuya Scramble Square",
			TimeZone: "Asia/Tokyo",
		},
	}
	booking.StartsAt.Time = time.Date(2026, 9, 11, 9, 0, 0, 0, loc)
	booking.EndsAt.Time = time.Date(2026, 9, 11, 20, 0, 0, 0, loc)
	return booking
}

func TestCompactBookingCarriesTimezone(t *testing.T) {
	got := compactBookingFromModel(tokyoBooking(t))

	if got.Timezone != "Asia/Tokyo" {
		t.Errorf("timezone = %q, want %q", got.Timezone, "Asia/Tokyo")
	}
	if got.StartTime != "09:00" || got.EndTime != "20:00" {
		t.Errorf("local times = %s-%s, want 09:00-20:00", got.StartTime, got.EndTime)
	}
	if got.StartsAt != "2026-09-11T09:00:00+09:00" {
		t.Errorf("starts_at = %q", got.StartsAt)
	}
	if got.EndsAt != "2026-09-11T20:00:00+09:00" {
		t.Errorf("ends_at = %q", got.EndsAt)
	}
	if got.StartsAtUTC != "2026-09-11T00:00:00Z" {
		t.Errorf("starts_at_utc = %q", got.StartsAtUTC)
	}
	if got.EndsAtUTC != "2026-09-11T11:00:00Z" {
		t.Errorf("ends_at_utc = %q", got.EndsAtUTC)
	}
	if got.TimezoneWarning != "" {
		t.Errorf("unexpected timezone warning: %q", got.TimezoneWarning)
	}
}

func TestCompactBookingWarnsWhenTimezoneUnknown(t *testing.T) {
	booking := tokyoBooking(t)
	booking.TimeZone = ""
	booking.Location.TimeZone = ""

	got := compactBookingFromModel(booking)

	if got.TimezoneWarning == "" {
		t.Error("expected a timezone warning when the zone is unknown")
	}
	if got.StartsAt != "" || got.EndsAt != "" || got.StartsAtUTC != "" || got.EndsAtUTC != "" {
		t.Errorf("expected timestamps to be omitted, got %+v", got)
	}
}
