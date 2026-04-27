package app

import (
	"testing"
	"time"

	"github.com/dvcrn/mcp-server-wework/internal/wework"
)

func TestParseDateSelection(t *testing.T) {
	t.Run("single date", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 1 || dates[0].Format("2006-01-02") != "2026-04-06" {
			t.Fatalf("unexpected dates: %#v", dates)
		}
	})

	t.Run("comma separated dates", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06,2026-04-08")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 2 {
			t.Fatalf("expected 2 dates, got %d", len(dates))
		}
	})

	t.Run("date range", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06~2026-04-08")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 3 {
			t.Fatalf("expected 3 dates, got %d", len(dates))
		}
		if dates[2].Format("2006-01-02") != "2026-04-08" {
			t.Fatalf("unexpected end date: %s", dates[2].Format("2006-01-02"))
		}
	})
}

func TestBuildCancelRequest(t *testing.T) {
	booking := &wework.Booking{
		UUID:     "booking-uuid",
		StartsAt: wework.CustomTime{Time: time.Date(2026, 4, 6, 8, 30, 0, 0, time.UTC)},
		EndsAt:   wework.CustomTime{Time: time.Date(2026, 4, 6, 20, 0, 0, 0, time.UTC)},
		CreditOrder: &wework.CreditOrder{
			Price: "2",
		},
		Reservable: &wework.SharedWorkspace{
			UUID:     "space-uuid",
			TypeName: "PrivateOffice",
			Location: &wework.SharedWorkspaceLocation{
				UUID:       "location-uuid",
				Name:       "WeWork Bryant Park",
				SourceType: 7,
				Address: wework.Address{
					Line1: "54 W 40th St",
				},
			},
		},
	}

	request, err := buildCancelRequest(booking, CancelBookingInput{BookingUUID: booking.UUID})
	if err != nil {
		t.Fatalf("buildCancelRequest returned error: %v", err)
	}
	if request.BookingID != "booking-uuid" {
		t.Fatalf("unexpected BookingID: %s", request.BookingID)
	}
	if request.BookingLocationType != 7 {
		t.Fatalf("unexpected BookingLocationType: %d", request.BookingLocationType)
	}
	if request.BookingType != cancelBookingTypePrivateOffice {
		t.Fatalf("unexpected BookingType: %d", request.BookingType)
	}
	if request.LocationID != "location-uuid" {
		t.Fatalf("unexpected LocationID: %s", request.LocationID)
	}
	if request.ReservableID != "space-uuid" {
		t.Fatalf("unexpected ReservableID: %s", request.ReservableID)
	}
}
