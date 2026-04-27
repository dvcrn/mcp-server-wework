package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/sahilm/fuzzy"

	"github.com/dvcrn/mcp-server-wework/internal/tzdate"
	"github.com/dvcrn/mcp-server-wework/internal/wework"
)

const (
	platformWeb = 1
	platformIOS = 2

	cancelBookingTypeConferenceRoom = 0
	cancelBookingTypePrivateOffice  = 2
	cancelBookingTypeSharedDesk     = 4
)

type Service struct {
	mu       sync.Mutex
	username string
	password string
	client   *wework.WeWork
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) clientForRequest() (*wework.WeWork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}

	username := strings.TrimSpace(firstNonEmpty(s.username, os.Getenv("WEWORK_USERNAME")))
	password := strings.TrimSpace(firstNonEmpty(s.password, os.Getenv("WEWORK_PASSWORD")))
	if username == "" || password == "" {
		return nil, fmt.Errorf("WEWORK_USERNAME and WEWORK_PASSWORD must be set in the environment")
	}

	auth, err := wework.NewWeWorkAuth(username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create WeWork auth client: %w", err)
	}

	login, _, err := auth.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	s.client = wework.NewWeWork(login.A0token)
	return s.client, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type LocationsInput struct {
	City string `json:"city"`
}

type DesksInput struct {
	LocationUUID string `json:"location_uuid,omitempty"`
	City         string `json:"city,omitempty"`
	Date         string `json:"date,omitempty"`
}

type BookingsInput struct {
	Past      bool   `json:"past,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

type BookInput struct {
	LocationUUID string `json:"location_uuid,omitempty"`
	City         string `json:"city,omitempty"`
	Name         string `json:"name,omitempty"`
	Date         string `json:"date"`
}

type QuoteInput struct {
	LocationUUID string `json:"location_uuid,omitempty"`
	City         string `json:"city,omitempty"`
	Name         string `json:"name,omitempty"`
	Date         string `json:"date"`
}

type InfoInput struct {
	LocationUUID  string `json:"location_uuid,omitempty"`
	City          string `json:"city,omitempty"`
	Name          string `json:"name,omitempty"`
	AmenitiesOnly bool   `json:"amenities_only,omitempty"`
}

type MeInput struct {
	IncludeBootstrap bool `json:"include_bootstrap,omitempty"`
}

type CalendarInput struct{}

type CancelBookingInput struct {
	BookingUUID         string `json:"booking_uuid"`
	BookingID           string `json:"booking_id,omitempty"`
	BookingLocationType *int   `json:"booking_location_type,omitempty"`
	ReservableID        string `json:"reservable_id,omitempty"`
	LocationID          string `json:"location_id,omitempty"`
	BookingType         *int   `json:"booking_type,omitempty"`
	ReservationID       string `json:"reservation_id,omitempty"`
	IsOnDemand          bool   `json:"is_on_demand,omitempty"`
	PlatformType        int    `json:"platform_type,omitempty"`
}

type AvailableSpace struct {
	Location        string `json:"location"`
	ReservableID    string `json:"reservable_id"`
	LocationID      string `json:"location_id"`
	Available       int    `json:"available"`
	ReservableType  string `json:"reservable_type,omitempty"`
	ReservableName  string `json:"reservable_name,omitempty"`
	ReservableFloor string `json:"reservable_floor,omitempty"`
}

type CompactBooking struct {
	UUID           string                       `json:"uuid"`
	Date           string                       `json:"date"`
	StartTime      string                       `json:"start_time"`
	EndTime        string                       `json:"end_time"`
	LocationName   string                       `json:"location_name"`
	LocationUUID   string                       `json:"location_uuid"`
	Address        string                       `json:"address"`
	City           string                       `json:"city"`
	Credits        string                       `json:"credits"`
	ReservableUUID string                       `json:"reservable_uuid,omitempty"`
	ReservableType string                       `json:"reservable_type,omitempty"`
	Cancellation   *wework.CancelBookingRequest `json:"cancellation,omitempty"`
}

type BookResult struct {
	Date          string                  `json:"date"`
	SpaceUUID     string                  `json:"space_uuid,omitempty"`
	LocationUUID  string                  `json:"location_uuid,omitempty"`
	LocationName  string                  `json:"location_name,omitempty"`
	BookingStatus *wework.BookingResponse `json:"booking,omitempty"`
	Error         string                  `json:"error,omitempty"`
}

type QuoteResult struct {
	Date         string                `json:"date"`
	SpaceUUID    string                `json:"space_uuid,omitempty"`
	LocationUUID string                `json:"location_uuid,omitempty"`
	LocationName string                `json:"location_name,omitempty"`
	Quote        *wework.QuoteResponse `json:"quote,omitempty"`
	Error        string                `json:"error,omitempty"`
}

type LocationsResult struct {
	Items []wework.GeoLocation `json:"items"`
}

type DesksResult struct {
	Items []AvailableSpace `json:"items"`
}

type BookingsResult struct {
	Items []CompactBooking `json:"items"`
}

type BookResults struct {
	Items []BookResult `json:"items"`
}

type QuoteResults struct {
	Items []QuoteResult `json:"items"`
}

type CalendarOutput struct {
	BookingsCount int    `json:"bookings_count"`
	ICS           string `json:"ics"`
}

type CancelBookingOutput struct {
	BookingUUID string                      `json:"booking_uuid"`
	Request     wework.CancelBookingRequest `json:"request"`
	Response    map[string]any              `json:"response"`
}

func (s *Service) Locations(ctx context.Context, input LocationsInput) (LocationsResult, error) {
	_ = ctx
	if strings.TrimSpace(input.City) == "" {
		return LocationsResult{}, fmt.Errorf("city is required")
	}
	ww, err := s.clientForRequest()
	if err != nil {
		return LocationsResult{}, err
	}
	res, err := ww.GetLocationsByGeo(input.City)
	if err != nil {
		return LocationsResult{}, err
	}
	return LocationsResult{Items: res.LocationsByGeo}, nil
}

func (s *Service) Desks(ctx context.Context, input DesksInput) (DesksResult, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return DesksResult{}, err
	}

	if strings.TrimSpace(input.LocationUUID) == "" && strings.TrimSpace(input.City) == "" {
		return DesksResult{}, fmt.Errorf("location_uuid or city is required")
	}

	date := input.Date
	if strings.TrimSpace(date) == "" {
		date = time.Now().Format("2006-01-02")
	}

	locationUUIDs, _, err := resolveLocationUUIDsForDesks(ww, input.LocationUUID, input.City)
	if err != nil {
		return DesksResult{}, err
	}

	dateParsed, err := tzdate.ParseInTimezone("2006-01-02", date, "Local")
	if err != nil {
		return DesksResult{}, err
	}

	resp, err := ww.GetAvailableSpaces(dateParsed, locationUUIDs)
	if err != nil {
		return DesksResult{}, err
	}

	rows := make([]AvailableSpace, 0, len(resp.Response.Workspaces))
	for _, space := range resp.Response.Workspaces {
		rows = append(rows, AvailableSpace{
			Location:        space.Location.Name,
			ReservableID:    space.UUID,
			LocationID:      space.Location.UUID,
			Available:       space.Seat.Available,
			ReservableType:  space.ReservableTypeName(),
			ReservableName:  space.ReservableName(),
			ReservableFloor: space.ReservableFloorName(),
		})
	}

	return DesksResult{Items: rows}, nil
}

func (s *Service) Bookings(ctx context.Context, input BookingsInput) (BookingsResult, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return BookingsResult{}, err
	}

	var bookings []*wework.Booking
	if input.Past {
		if input.StartDate != "" || input.EndDate != "" {
			var start, end time.Time
			if input.StartDate != "" {
				start, err = time.Parse("2006-01-02", input.StartDate)
				if err != nil {
					return BookingsResult{}, fmt.Errorf("invalid start_date: %w", err)
				}
			} else {
				start = time.Now().AddDate(0, 0, -30)
			}
			if input.EndDate != "" {
				end, err = time.Parse("2006-01-02", input.EndDate)
				if err != nil {
					return BookingsResult{}, fmt.Errorf("invalid end_date: %w", err)
				}
			} else {
				end = time.Now()
			}
			bookings, err = ww.GetPastBookingsWithDates(start, end)
		} else {
			bookings, err = ww.GetPastBookings()
		}
	} else {
		bookings, err = ww.GetUpcomingBookings()
	}
	if err != nil {
		return BookingsResult{}, err
	}

	rows := make([]CompactBooking, 0, len(bookings))
	for _, booking := range bookings {
		rows = append(rows, compactBookingFromModel(booking))
	}
	return BookingsResult{Items: rows}, nil
}

func (s *Service) Book(ctx context.Context, input BookInput) (BookResults, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return BookResults{}, err
	}

	targetLocationUUID, err := resolveLocationUUID(ww, input.City, input.Name, input.LocationUUID)
	if err != nil {
		return BookResults{}, err
	}

	dates, err := parseDateSelection(input.Date)
	if err != nil {
		return BookResults{}, err
	}

	results := make([]BookResult, 0, len(dates))
	for _, bookingDate := range dates {
		row := BookResult{Date: bookingDate.Format("2006-01-02")}

		spaces, err := ww.GetAvailableSpaces(bookingDate, []string{targetLocationUUID})
		if err != nil {
			row.Error = fmt.Sprintf("error getting spaces: %v", err)
			results = append(results, row)
			continue
		}
		if len(spaces.Response.Workspaces) == 0 {
			row.Error = "no spaces found"
			results = append(results, row)
			continue
		}
		if len(spaces.Response.Workspaces) > 1 {
			row.Error = "multiple spaces found, please specify a more specific location"
			results = append(results, row)
			continue
		}

		space := spaces.Response.Workspaces[0]
		row.SpaceUUID = space.UUID
		row.LocationUUID = space.Location.UUID
		row.LocationName = space.Location.Name

		bookRes, err := ww.PostBooking(bookingDate, &space)
		if err != nil {
			row.Error = fmt.Sprintf("booking failed: %v", err)
		} else {
			row.BookingStatus = bookRes
		}
		results = append(results, row)
	}

	return BookResults{Items: results}, nil
}

func (s *Service) Quote(ctx context.Context, input QuoteInput) (QuoteResults, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return QuoteResults{}, err
	}

	targetLocationUUID, err := resolveLocationUUID(ww, input.City, input.Name, input.LocationUUID)
	if err != nil {
		return QuoteResults{}, err
	}

	dates, err := parseDateSelection(input.Date)
	if err != nil {
		return QuoteResults{}, err
	}

	results := make([]QuoteResult, 0, len(dates))
	for _, bookingDate := range dates {
		row := QuoteResult{Date: bookingDate.Format("2006-01-02")}

		spaces, err := ww.GetAvailableSpaces(bookingDate, []string{targetLocationUUID})
		if err != nil {
			row.Error = fmt.Sprintf("error getting spaces: %v", err)
			results = append(results, row)
			continue
		}
		if len(spaces.Response.Workspaces) == 0 {
			row.Error = "no spaces found"
			results = append(results, row)
			continue
		}
		if len(spaces.Response.Workspaces) > 1 {
			row.Error = "multiple spaces found, please specify a more specific location"
			results = append(results, row)
			continue
		}

		space := spaces.Response.Workspaces[0]
		row.SpaceUUID = space.UUID
		row.LocationUUID = space.Location.UUID
		row.LocationName = space.Location.Name

		quote, err := ww.GetBookingQuote(bookingDate, &space)
		if err != nil {
			row.Error = fmt.Sprintf("failed to get booking quote: %v", err)
		} else {
			row.Quote = quote
		}
		results = append(results, row)
	}

	return QuoteResults{Items: results}, nil
}

func (s *Service) Info(ctx context.Context, input InfoInput) (*wework.LocationFeaturesResponse, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}

	locationUUID := input.LocationUUID
	if locationUUID == "" {
		locationUUID, err = resolveLocationUUID(ww, input.City, input.Name, "")
		if err != nil {
			return nil, err
		}
	}

	return ww.GetLocationFeatures(locationUUID, input.AmenitiesOnly)
}

func (s *Service) Me(ctx context.Context, input MeInput) (any, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}

	profile, err := ww.GetUserProfile()
	if err != nil {
		return nil, err
	}
	if !input.IncludeBootstrap {
		return profile, nil
	}

	bootstrap, err := ww.GetBootstrap()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"userProfile": profile,
		"bootstrap":   bootstrap,
	}, nil
}

func (s *Service) Calendar(ctx context.Context, input CalendarInput) (CalendarOutput, error) {
	_ = ctx
	_ = input
	ww, err := s.clientForRequest()
	if err != nil {
		return CalendarOutput{}, err
	}

	pastBookings, err := ww.GetPastBookings()
	if err != nil {
		return CalendarOutput{}, err
	}
	upcomingBookings, err := ww.GetUpcomingBookings()
	if err != nil {
		return CalendarOutput{}, err
	}

	if len(pastBookings) > 10 {
		pastBookings = pastBookings[:10]
	}
	allBookings := append(pastBookings, upcomingBookings...)

	cal := ics.NewCalendar()
	cal.SetProductId("-//WeWork Calendar//mcp-server-wework//")
	cal.SetVersion("2.0")

	for _, booking := range allBookings {
		if booking == nil || booking.Reservable == nil || booking.Reservable.Location == nil {
			continue
		}
		event := cal.AddEvent(booking.UUID)
		event.SetSummary(fmt.Sprintf("WeWork: %s", booking.Reservable.Location.Name))
		event.SetProperty(ics.ComponentProperty("DTSTART;TZID="+booking.Reservable.Location.TimeZone), booking.StartsAt.Format("20060102"))
		event.SetProperty(ics.ComponentProperty("DTEND;TZID="+booking.Reservable.Location.TimeZone), booking.StartsAt.Format("20060102"))
		event.SetProperty(ics.ComponentProperty("TZID"), booking.Reservable.Location.TimeZone)
		event.SetProperty("X-MICROSOFT-CDO-ALLDAYEVENT", "TRUE")
		event.SetProperty("X-MICROSOFT-CDO-BUSYSTATUS", "FREE")
		event.SetProperty("X-MICROSOFT-CDO-IMPORTANCE", "1")
		event.SetProperty("X-MICROSOFT-DISALLOW-COUNTER", "TRUE")
		event.SetProperty("X-APPLE-TRAVEL-ADVISORY-BEHAVIOR", "DISABLED")
		event.SetProperty("X-MOZ-LASTACK", "0")
		event.SetProperty("TRANSP", "TRANSPARENT")
		event.SetProperty("URL", "https://members.wework.com/workplaceone/content2/your-bookings")
		event.SetLocation(booking.Reservable.Location.Address.Line1)
		event.SetDescription(fmt.Sprintf(
			"WeWork Booking Details:\nLocation: %s\nAddress: %s\nTime: %s - %s\nBooking ID: %s",
			booking.Reservable.Location.Name,
			booking.Reservable.Location.Address.Line1,
			booking.StartsAt.Format("03:04 PM"),
			booking.EndsAt.Format("03:04 PM"),
			booking.UUID,
		))
	}

	buf := new(bytes.Buffer)
	if err := cal.SerializeTo(buf); err != nil {
		return CalendarOutput{}, err
	}

	return CalendarOutput{BookingsCount: len(allBookings), ICS: buf.String()}, nil
}

func (s *Service) CancelBooking(ctx context.Context, input CancelBookingInput) (CancelBookingOutput, error) {
	_ = ctx
	if strings.TrimSpace(input.BookingUUID) == "" {
		return CancelBookingOutput{}, fmt.Errorf("booking_uuid is required")
	}

	ww, err := s.clientForRequest()
	if err != nil {
		return CancelBookingOutput{}, err
	}

	bookings, err := ww.GetUpcomingBookings()
	if err != nil {
		return CancelBookingOutput{}, fmt.Errorf("failed to fetch upcoming bookings: %w", err)
	}

	var target *wework.Booking
	for _, booking := range bookings {
		if booking != nil && booking.UUID == input.BookingUUID {
			target = booking
			break
		}
	}
	if target == nil {
		return CancelBookingOutput{}, fmt.Errorf("no upcoming booking found with uuid %s", input.BookingUUID)
	}

	request, err := buildCancelRequest(target, input)
	if err != nil {
		return CancelBookingOutput{}, err
	}

	platformType := input.PlatformType
	if platformType == 0 {
		platformType = platformIOS
	}

	response, err := ww.CancelBooking(request, input.IsOnDemand, platformType)
	if err != nil {
		return CancelBookingOutput{}, err
	}

	return CancelBookingOutput{
		BookingUUID: input.BookingUUID,
		Request:     request,
		Response:    response,
	}, nil
}

func resolveLocationUUIDsForDesks(ww *wework.WeWork, locationUUID, city string) ([]string, string, error) {
	if city != "" {
		cities, err := ww.GetCities()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get cities: %w", err)
		}
		matchedCities, err := wework.FindCityByFuzzyName(city, cities)
		if err != nil {
			return nil, "", err
		}
		var allLocations []wework.GeoLocation
		for _, matchedCity := range matchedCities {
			res, err := ww.GetLocationsByGeo(matchedCity.Name)
			if err != nil {
				return nil, "", fmt.Errorf("failed to get locations for %s: %w", matchedCity.Name, err)
			}
			allLocations = append(allLocations, res.LocationsByGeo...)
		}
		if len(allLocations) == 0 {
			return nil, "", fmt.Errorf("no locations found in matched cities")
		}
		locationUUIDs := make([]string, 0, len(allLocations))
		for _, location := range allLocations {
			locationUUIDs = append(locationUUIDs, location.UUID)
		}
		return locationUUIDs, allLocations[0].TimeZone, nil
	}

	locationUUIDs := strings.Split(locationUUID, ",")
	locResp, err := ww.GetSpacesByUUIDs([]string{locationUUIDs[0]})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get location details: %w", err)
	}
	if len(locResp.Response.Workspaces) == 0 {
		return nil, "", fmt.Errorf("no spaces found for location uuid %s", locationUUIDs[0])
	}
	return locationUUIDs, locResp.Response.Workspaces[0].Location.TimeZone, nil
}

func resolveLocationUUID(ww *wework.WeWork, city, name, locationUUID string) (string, error) {
	if locationUUID != "" {
		return locationUUID, nil
	}
	if city == "" || name == "" {
		return "", fmt.Errorf("either location_uuid or both city and name are required")
	}

	cities, err := ww.GetCities()
	if err != nil {
		return "", fmt.Errorf("failed to get cities: %w", err)
	}

	matchedCities, err := wework.FindCityByFuzzyName(city, cities)
	if err != nil {
		return "", err
	}

	var allLocations []wework.GeoLocation
	for _, matchedCity := range matchedCities {
		res, err := ww.GetLocationsByGeo(matchedCity.Name)
		if err != nil {
			return "", fmt.Errorf("failed to get locations for %s: %w", matchedCity.Name, err)
		}
		allLocations = append(allLocations, res.LocationsByGeo...)
	}

	if len(allLocations) == 0 {
		return "", fmt.Errorf("no locations found in city %s", city)
	}

	return findLocationByFuzzyName(name, allLocations)
}

func findLocationByFuzzyName(name string, locations []wework.GeoLocation) (string, error) {
	var names []string
	for _, loc := range locations {
		names = append(names, loc.Name)
	}
	matches := fuzzy.Find(name, names)
	if len(matches) == 0 {
		return "", fmt.Errorf("no location found matching %q", name)
	}
	if len(matches) > 1 {
		var matchNames []string
		for _, match := range matches {
			matchNames = append(matchNames, match.Str)
		}
		return "", fmt.Errorf("multiple locations found: %s", strings.Join(matchNames, ", "))
	}
	return locations[matches[0].Index].UUID, nil
}

func parseDateSelection(input string) ([]time.Time, error) {
	date := strings.TrimSpace(input)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	var dates []time.Time
	if strings.Contains(date, "~") {
		parts := strings.Split(date, "~")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid date range format; expected YYYY-MM-DD~YYYY-MM-DD")
		}
		startDate, err := tzdate.ParseInTimezone("2006-01-02", strings.TrimSpace(parts[0]), "Local")
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %w", err)
		}
		endDate, err := tzdate.ParseInTimezone("2006-01-02", strings.TrimSpace(parts[1]), "Local")
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %w", err)
		}
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
		return dates, nil
	}
	if strings.Contains(date, ",") {
		for _, part := range strings.Split(date, ",") {
			parsed, err := tzdate.ParseInTimezone("2006-01-02", strings.TrimSpace(part), "Local")
			if err != nil {
				return nil, fmt.Errorf("invalid date %q: %w", part, err)
			}
			dates = append(dates, parsed)
		}
		return dates, nil
	}
	parsed, err := tzdate.ParseInTimezone("2006-01-02", date, "Local")
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	return []time.Time{parsed}, nil
}

func compactBookingFromModel(booking *wework.Booking) CompactBooking {
	result := CompactBooking{}
	if booking == nil {
		return result
	}

	result.UUID = booking.UUID
	result.Date = booking.StartsAt.Time.Format("2006-01-02")
	result.StartTime = booking.StartsAt.Time.Format("15:04")
	result.EndTime = booking.EndsAt.Time.Format("15:04")
	if booking.CreditOrder != nil {
		result.Credits = booking.CreditOrder.Price
	}
	if booking.Reservable != nil {
		result.ReservableUUID = booking.Reservable.UUID
		result.ReservableType = booking.Reservable.TypeName
		if booking.Reservable.Location != nil {
			result.LocationName = booking.Reservable.Location.Name
			result.LocationUUID = booking.Reservable.Location.UUID
			result.Address = booking.Reservable.Location.Address.Line1
			result.City = booking.Reservable.Location.Address.City
		}
		if cancellation, err := buildCancelRequest(booking, CancelBookingInput{BookingUUID: booking.UUID}); err == nil {
			result.Cancellation = &cancellation
		}
	}
	return result
}

func buildCancelRequest(booking *wework.Booking, input CancelBookingInput) (wework.CancelBookingRequest, error) {
	if booking == nil || booking.Reservable == nil || booking.Reservable.Location == nil {
		return wework.CancelBookingRequest{}, fmt.Errorf("booking is missing reservable location data")
	}

	bookingLocationType := booking.Reservable.Location.SourceType
	if input.BookingLocationType != nil {
		bookingLocationType = *input.BookingLocationType
	}

	bookingID := input.BookingID
	if bookingID == "" {
		bookingID = booking.UUID
		if booking.IsFromKube && booking.KubeBookingExternalReference != "" {
			bookingID = booking.KubeBookingExternalReference
		}
	}

	bookingType := cancelBookingTypeSharedDesk
	if input.BookingType != nil {
		bookingType = *input.BookingType
	} else {
		bookingType = cancelBookingTypeFromTypeName(booking.Reservable.TypeName)
	}

	reservationID := firstNonEmpty(input.ReservationID, booking.UUID)
	locationID := firstNonEmpty(input.LocationID, booking.Reservable.Location.UUID)
	reservableID := firstNonEmpty(input.ReservableID, booking.Reservable.UUID)
	creditsUsed := ""
	if booking.CreditOrder != nil {
		creditsUsed = booking.CreditOrder.Price
	}

	return wework.CancelBookingRequest{
		BookingID:           bookingID,
		BookingLocationType: bookingLocationType,
		ReservableID:        reservableID,
		StartTime:           booking.StartsAt.Time.Format(time.RFC3339),
		EndTime:             booking.EndsAt.Time.Format(time.RFC3339),
		CreditsUsed:         creditsUsed,
		LocationID:          locationID,
		MailParams:          cancelMailDataFromBooking(booking),
		BookingType:         bookingType,
		ReservationID:       reservationID,
	}, nil
}

func cancelMailDataFromBooking(booking *wework.Booking) wework.CancelMailData {
	locationName := ""
	address := ""
	workspaceType := ""
	if booking != nil && booking.Reservable != nil {
		workspaceType = booking.Reservable.TypeName
		if booking.Reservable.Location != nil {
			locationName = booking.Reservable.Location.Name
			address = booking.Reservable.Location.Address.Line1
		}
	}
	locationAddress := strings.TrimSpace(strings.TrimSpace(locationName + " " + address))
	return wework.CancelMailData{
		WorkspaceType:      workspaceType,
		DayFormatted:       booking.StartsAt.Time.Format("Monday, January 2"),
		StartTimeFormatted: booking.StartsAt.Time.Format("3:04 PM"),
		EndTimeFormatted:   booking.EndsAt.Time.Format("3:04 PM"),
		FloorAddress:       "",
		LocationAddress:    locationAddress,
	}
}

func cancelBookingTypeFromTypeName(typeName string) int {
	switch typeName {
	case "ConferenceRoom":
		return cancelBookingTypeConferenceRoom
	case "PrivateOffice":
		return cancelBookingTypePrivateOffice
	default:
		return cancelBookingTypeSharedDesk
	}
}
