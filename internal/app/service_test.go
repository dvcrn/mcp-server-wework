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

func TestBuildMarkFavoriteLocationRequestDefaults(t *testing.T) {
	locationTypeZero := 0
	locationAccountTypeZero := 0
	tests := []struct {
		name                string
		input               MarkFavoriteLocationInput
		spaceType           int
		locationType        int
		locationAccountType int
		reservableUUID      string
		inventoryName       string
	}{
		{
			name: "desk",
			input: MarkFavoriteLocationInput{
				LocationID:    "5281345f-94bd-4ad0-83e0-472c6829fb72",
				SpaceType:     0,
				InventoryName: "1 Mark Sq",
			},
			spaceType:           0,
			locationType:        2,
			locationAccountType: 2,
			inventoryName:       "1 Mark Sq",
		},
		{
			name: "private office",
			input: MarkFavoriteLocationInput{
				LocationID:     "dbccfd91-4873-40e3-a9b9-89b7a7b68549",
				SpaceType:      1,
				ReservableUUID: "dc454f19-33c8-4e3a-b149-6269a8314749",
				InventoryName:  "06B113",
			},
			spaceType:           1,
			locationType:        2,
			locationAccountType: 2,
			reservableUUID:      "dc454f19-33c8-4e3a-b149-6269a8314749",
			inventoryName:       "06B113",
		},
		{
			name: "room",
			input: MarkFavoriteLocationInput{
				LocationID:     "dbccfd91-4873-40e3-a9b9-89b7a7b68549",
				SpaceType:      2,
				ReservableUUID: "1222573b-2d84-4633-916f-7b6027e7e48d",
				InventoryName:  "5A",
			},
			spaceType:           2,
			locationType:        2,
			locationAccountType: 2,
			reservableUUID:      "1222573b-2d84-4633-916f-7b6027e7e48d",
			inventoryName:       "5A",
		},
		{
			name: "explicit zero location types",
			input: MarkFavoriteLocationInput{
				LocationID:          "dbccfd91-4873-40e3-a9b9-89b7a7b68549",
				SpaceType:           0,
				LocationType:        &locationTypeZero,
				LocationAccountType: &locationAccountTypeZero,
				InventoryName:       "Bangkok",
			},
			spaceType:           0,
			locationType:        0,
			locationAccountType: 0,
			inventoryName:       "Bangkok",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := buildMarkFavoriteLocationRequest(tc.input)

			if request.LocationID != tc.input.LocationID {
				t.Fatalf("unexpected LocationID: %s", request.LocationID)
			}
			if request.SpaceType != tc.spaceType {
				t.Fatalf("unexpected SpaceType: %d", request.SpaceType)
			}
			if request.LocationType != tc.locationType {
				t.Fatalf("unexpected LocationType: %d", request.LocationType)
			}
			if request.LocationAccountType != tc.locationAccountType {
				t.Fatalf("unexpected LocationAccountType: %d", request.LocationAccountType)
			}
			if request.PlatformType != "WEB" {
				t.Fatalf("unexpected PlatformType: %s", request.PlatformType)
			}
			if request.ApplicationType != "WorkplaceOne" {
				t.Fatalf("unexpected ApplicationType: %s", request.ApplicationType)
			}
			if request.ReservableUUID != tc.reservableUUID {
				t.Fatalf("unexpected ReservableUUID: %s", request.ReservableUUID)
			}
			if request.InventoryName != tc.inventoryName {
				t.Fatalf("unexpected InventoryName: %s", request.InventoryName)
			}
		})
	}
}
