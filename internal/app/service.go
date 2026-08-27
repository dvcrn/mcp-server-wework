package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/dvcrn/wework-cli/pkg/wework"
	"github.com/sahilm/fuzzy"

	"github.com/dvcrn/mcp-server-wework/internal/tzdate"
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
	BookingUUID string `json:"booking_uuid"`
}

type FavoritesInput struct {
	SpaceType int `json:"space_type,omitempty"`
}

type AddFavoriteInput struct {
	LocationUUID string `json:"location_uuid,omitempty"`
	City         string `json:"city,omitempty"`
	Name         string `json:"name,omitempty"`
	SpaceType    int    `json:"space_type,omitempty"`
}

type RemoveFavoriteInput struct {
	LocationUUID string `json:"location_uuid,omitempty"`
	City         string `json:"city,omitempty"`
	Name         string `json:"name,omitempty"`
	Hmy          int    `json:"hmy,omitempty"`
}

// Constants for favorite mutations, matching the values the WeWork web/iOS app
// sends. Mirrors the wework-cli favorites commands.
const (
	favoritePlatformType    = "iOS_APP"
	favoriteApplicationType = "WorkplaceOne"
	favoriteLocationType    = 2
	favoriteAccountType     = 4
	favoriteMaxSpaceType    = 3
)

type PrintQueueInput struct {
	JobIDs string `json:"job_ids,omitempty"`
}

type AddPrintJobInput struct {
	FileName        string `json:"file_name"`
	FileBase64      string `json:"file_base64"`
	FileContentType string `json:"file_content_type,omitempty"`
	JobName         string `json:"job_name,omitempty"`
	Copies          int    `json:"copies,omitempty"`
	Orientation     string `json:"orientation,omitempty"`
	ColorMode       string `json:"color_mode,omitempty"`
	Sides           string `json:"sides,omitempty"`
	ForceMediaSize  string `json:"force_media_size,omitempty"`
}

type FavoritesResult struct {
	Items   []wework.FavoriteLocation `json:"items"`
	Recents []wework.FavoriteLocation `json:"recents,omitempty"`
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

// CompactBooking reports booking times in the location's timezone. date,
// start_time and end_time are local wall clock; starts_at/ends_at repeat them as
// RFC 3339 timestamps with offset, and starts_at_utc/ends_at_utc give the absolute
// instants for ordering and comparison. When the location's timezone cannot be
// resolved the timestamp fields are omitted and timezone_warning says so, rather
// than emitting a zoneless time that looks interpretable.
type CompactBooking struct {
	UUID            string `json:"uuid"`
	Date            string `json:"date"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	StartsAt        string `json:"starts_at,omitempty"`
	EndsAt          string `json:"ends_at,omitempty"`
	StartsAtUTC     string `json:"starts_at_utc,omitempty"`
	EndsAtUTC       string `json:"ends_at_utc,omitempty"`
	Timezone        string `json:"timezone,omitempty"`
	TimezoneWarning string `json:"timezone_warning,omitempty"`
	LocationName    string `json:"location_name"`
	LocationUUID    string `json:"location_uuid"`
	Address         string `json:"address"`
	City            string `json:"city"`
	Credits         string `json:"credits"`
	ReservableUUID  string `json:"reservable_uuid,omitempty"`
	ReservableType  string `json:"reservable_type,omitempty"`
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
	BookingUUID string                        `json:"booking_uuid"`
	Request     *wework.CancelBookingRequest  `json:"request,omitempty"`
	Response    *wework.CancelBookingResponse `json:"response,omitempty"`
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
			ReservableType:  reservableTypeName(space),
			ReservableName:  reservableName(space),
			ReservableFloor: reservableFloorName(space),
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
		if booking == nil || booking.Location == nil {
			continue
		}
		event := cal.AddEvent(booking.BookingID)
		event.SetSummary(fmt.Sprintf("WeWork: %s", booking.Location.Name))
		event.SetProperty(ics.ComponentProperty("DTSTART;TZID="+booking.Location.TimeZone), booking.StartsAt.Format("20060102"))
		event.SetProperty(ics.ComponentProperty("DTEND;TZID="+booking.Location.TimeZone), booking.StartsAt.Format("20060102"))
		event.SetProperty(ics.ComponentProperty("TZID"), booking.Location.TimeZone)
		event.SetProperty("X-MICROSOFT-CDO-ALLDAYEVENT", "TRUE")
		event.SetProperty("X-MICROSOFT-CDO-BUSYSTATUS", "FREE")
		event.SetProperty("X-MICROSOFT-CDO-IMPORTANCE", "1")
		event.SetProperty("X-MICROSOFT-DISALLOW-COUNTER", "TRUE")
		event.SetProperty("X-APPLE-TRAVEL-ADVISORY-BEHAVIOR", "DISABLED")
		event.SetProperty("X-MOZ-LASTACK", "0")
		event.SetProperty("TRANSP", "TRANSPARENT")
		event.SetProperty("URL", "https://members.wework.com/workplaceone/content2/your-bookings")
		event.SetLocation(booking.Location.Address.Line1)
		event.SetDescription(fmt.Sprintf(
			"WeWork Booking Details:\nLocation: %s\nAddress: %s\nTime: %s - %s\nBooking ID: %s",
			booking.Location.Name,
			booking.Location.Address.Line1,
			booking.StartsAt.Format("03:04 PM"),
			booking.EndsAt.Format("03:04 PM"),
			booking.BookingID,
		))
	}

	buf := new(bytes.Buffer)
	if err := cal.SerializeTo(buf); err != nil {
		return CalendarOutput{}, err
	}

	return CalendarOutput{BookingsCount: len(allBookings), ICS: buf.String()}, nil
}

// CancelBooking returns any (not CancelBookingOutput) so the MCP SDK skips
// output-schema generation and validation for this tool. The cancel endpoint's
// response body is a bare boolean, which can't be meaningfully schema-typed:
// reflecting wework.CancelBookingResponse.Raw (json.RawMessage) yields a
// ["null","array"] schema the boolean fails at runtime, and an `any` field
// reflects to a bare `true` schema that the MCP client rejects at load time.
// Mirrors the Me and Info handlers, which return any for the same reason.
func (s *Service) CancelBooking(ctx context.Context, input CancelBookingInput) (any, error) {
	_ = ctx
	if strings.TrimSpace(input.BookingUUID) == "" {
		return nil, fmt.Errorf("booking_uuid is required")
	}

	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}

	request, err := ww.BuildCancelBookingRequest(input.BookingUUID)
	if err != nil {
		return nil, err
	}

	response, err := ww.CancelBooking(input.BookingUUID)
	if err != nil {
		return nil, err
	}

	return CancelBookingOutput{
		BookingUUID: input.BookingUUID,
		Request:     request,
		Response:    response,
	}, nil
}

// Favorites lists the member's favorite (and recent) locations for a space type.
// space_type must be 0-3; the WeWork API rejects other values.
func (s *Service) Favorites(ctx context.Context, input FavoritesInput) (FavoritesResult, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return FavoritesResult{}, err
	}
	resp, err := ww.GetFavoriteLocations(input.SpaceType)
	if err != nil {
		return FavoritesResult{}, err
	}
	return FavoritesResult{Items: resp.FavoriteLocations, Recents: resp.RecentLocations}, nil
}

// AddFavorite favorites a location, identified by UUID or by city + name.
func (s *Service) AddFavorite(ctx context.Context, input AddFavoriteInput) (any, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}
	uuid, err := resolveLocationUUID(ww, input.City, input.Name, input.LocationUUID)
	if err != nil {
		return nil, err
	}
	return ww.MarkFavoriteLocation(favoriteRequest(uuid, input.SpaceType, false, 0))
}

// RemoveFavorite removes a location from the member's favorites. The API deletes
// by the favorite's numeric id (its "hmy"), so unless an exact hmy is given the
// location is resolved (by UUID or city + name) and matched against the current
// favorites across every space type to find the id(s) to delete.
func (s *Service) RemoveFavorite(ctx context.Context, input RemoveFavoriteInput) (any, error) {
	_ = ctx
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}

	if input.Hmy > 0 {
		return ww.MarkFavoriteLocation(favoriteRequest("", 0, true, input.Hmy))
	}

	uuid, err := resolveLocationUUID(ww, input.City, input.Name, input.LocationUUID)
	if err != nil {
		return nil, err
	}

	type match struct {
		id        int
		spaceType int
	}
	var matches []match
	for st := 0; st <= favoriteMaxSpaceType; st++ {
		favs, err := ww.GetFavoriteLocations(st)
		if err != nil {
			return nil, fmt.Errorf("failed to look up favorites (space_type %d): %w", st, err)
		}
		for _, f := range favs.FavoriteLocations {
			if f.LocationID == uuid && f.Hmy > 0 {
				matches = append(matches, match{id: f.Hmy, spaceType: st})
			}
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("location %s is not in your favorites", uuid)
	}

	var last map[string]any
	for _, m := range matches {
		last, err = ww.MarkFavoriteLocation(favoriteRequest("", m.spaceType, true, m.id))
		if err != nil {
			return nil, fmt.Errorf("failed to remove favorite (id %d): %w", m.id, err)
		}
	}
	return last, nil
}

// favoriteRequest builds the mark-as-favorite payload with the app's default
// platform/application/location classification.
func favoriteRequest(locationUUID string, spaceType int, remove bool, id int) wework.MarkFavoriteLocationRequest {
	return wework.MarkFavoriteLocationRequest{
		ID:                  id,
		LocationID:          locationUUID,
		SpaceType:           spaceType,
		IsDeleted:           remove,
		LocationType:        favoriteLocationType,
		LocationAccountType: favoriteAccountType,
		PlatformType:        favoritePlatformType,
		ApplicationType:     favoriteApplicationType,
	}
}

// PrintQueue returns the member's current print queue.
func (s *Service) PrintQueue(ctx context.Context, input PrintQueueInput) (*wework.PrintQueueResponse, error) {
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}
	return ww.GetPrintQueue(ctx, input.JobIDs)
}

// AddPrintJob uploads a base64-encoded document to the member's print queue.
func (s *Service) AddPrintJob(ctx context.Context, input AddPrintJobInput) (*wework.PrintJob, error) {
	if strings.TrimSpace(input.FileName) == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	fileBytes, err := decodeBase64Loose(input.FileBase64)
	if err != nil {
		return nil, fmt.Errorf("file_base64 is not valid base64: %w", err)
	}
	if len(fileBytes) == 0 {
		return nil, fmt.Errorf("file_base64 is required")
	}
	ww, err := s.clientForRequest()
	if err != nil {
		return nil, err
	}
	return ww.AddToPrintQueue(ctx, wework.AddPrintJobRequest{
		Copies:               input.Copies,
		ForceMediaSize:       input.ForceMediaSize,
		OrientationRequested: input.Orientation,
		PrintColorMode:       input.ColorMode,
		Sides:                input.Sides,
		JobName:              input.JobName,
		FileName:             input.FileName,
		FileContentType:      input.FileContentType,
		FileBytes:            fileBytes,
	})
}

// decodeBase64Loose decodes base64 that may arrive with surrounding or internal
// whitespace — LLMs often wrap long payloads at 76 columns like PEM — and may use
// either the standard or URL-safe alphabet, with or without padding.
func decodeBase64Loose(s string) ([]byte, error) {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f':
			continue
		default:
			b.WriteRune(r)
		}
	}
	cleaned := b.String()

	var err error
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	} {
		var decoded []byte
		if decoded, err = enc.DecodeString(cleaned); err == nil {
			return decoded, nil
		}
	}
	return nil, err
}

func reservableTypeName(space wework.Workspace) string {
	if space.Reservable == nil {
		return ""
	}
	if space.IsHybridSpace {
		return "HybridSpace"
	}
	if space.IsAffiliateCoworking {
		return "AffiliateCoworking"
	}
	if space.IsFranchiseCoworking {
		return "FranchiseCoworking"
	}
	return "Workspace"
}

func reservableName(space wework.Workspace) string {
	return space.Location.Name
}

func reservableFloorName(space wework.Workspace) string {
	return ""
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

	startsAt := booking.StartsAt.Time
	endsAt := booking.EndsAt.Time

	result.UUID = booking.BookingID
	result.Date = startsAt.Format("2006-01-02")
	result.StartTime = startsAt.Format("15:04")
	result.EndTime = endsAt.Format("15:04")
	result.Timezone = bookingTimezone(booking)
	if result.Timezone == "" {
		result.TimezoneWarning = "timezone unavailable for this booking; times are wall clock as reported by the API and cannot be resolved to an absolute instant"
	} else {
		result.StartsAt = startsAt.Format(time.RFC3339)
		result.EndsAt = endsAt.Format(time.RFC3339)
		result.StartsAtUTC = startsAt.UTC().Format(time.RFC3339)
		result.EndsAtUTC = endsAt.UTC().Format(time.RFC3339)
	}
	result.Credits = strconv.FormatFloat(booking.CreditCost, 'f', -1, 64)
	result.ReservableUUID = booking.SpaceID
	if booking.Location != nil {
		result.LocationName = booking.Location.Name
		result.LocationUUID = booking.Location.UUID
		result.Address = booking.Location.Address.Line1
		result.City = booking.Location.Address.City
	}
	return result
}

// bookingTimezone returns the IANA timezone the booking's times are expressed in,
// preferring the booking's own field and falling back to its location's.
func bookingTimezone(booking *wework.Booking) string {
	if booking.TimeZone != "" {
		return booking.TimeZone
	}
	if booking.Location != nil {
		return booking.Location.TimeZone
	}
	return ""
}
