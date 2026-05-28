package app

import (
	"encoding/json"
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

func TestDirectBookingPayload(t *testing.T) {
	spaceType := 4
	spaceTypeID := 0
	locationType := 2
	platformType := 1
	triggerCalendarEvent := true
	creditCharged := json.RawMessage(`0`)

	input := BookInput{
		StartTime:            "2026-06-07T05:00:00Z",
		EndTime:              "2026-06-07T22:59:00Z",
		LocationID:           "20242b8b-d507-44e4-942e-1d89814bbf38",
		WeWorkSpaceID:        "519f596a-99d4-11ea-bbbf-0abba6a90f13",
		SpaceID:              "15769",
		SpaceType:            &spaceType,
		SpaceTypeID:          &spaceTypeID,
		LocationType:         &locationType,
		ApplicationType:      "WorkplaceOne",
		PlatFormTypeEnum:     &platformType,
		UTCOffset:            "+01:00",
		TriggerCalendarEvent: &triggerCalendarEvent,
		CreditCharged:        &creditCharged,
		Currency:             "com.wework.credits",
	}

	payload := directBookingPayload(input)

	assertPayloadValue(t, payload, "StartTime", "2026-06-07T05:00:00Z")
	assertPayloadValue(t, payload, "EndTime", "2026-06-07T22:59:00Z")
	assertPayloadValue(t, payload, "LocationID", "20242b8b-d507-44e4-942e-1d89814bbf38")
	assertPayloadValue(t, payload, "WeWorkSpaceID", "519f596a-99d4-11ea-bbbf-0abba6a90f13")
	assertPayloadValue(t, payload, "SpaceID", "15769")
	assertPayloadValue(t, payload, "SpaceType", 4)
	assertPayloadValue(t, payload, "SpaceTypeID", 0)
	assertPayloadValue(t, payload, "LocationType", 2)
	assertPayloadValue(t, payload, "ApplicationType", "WorkplaceOne")
	assertPayloadValue(t, payload, "PlatFormTypeEnum", 1)
	assertPayloadValue(t, payload, "UTCOffset", "+01:00")
	assertPayloadValue(t, payload, "TriggerCalendarEvent", true)
	assertPayloadValue(t, payload, "CreditCharged", float64(0))
	assertPayloadValue(t, payload, "Currency", "com.wework.credits")
}

func assertPayloadValue(t *testing.T, payload map[string]any, key string, want any) {
	t.Helper()
	if got := payload[key]; got != want {
		t.Fatalf("payload[%s] = %#v, want %#v", key, got, want)
	}
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
