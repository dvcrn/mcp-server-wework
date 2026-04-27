package wework

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"
)

type WeWork struct {
	client     *BaseClient
	cities     []*CityDetailsResponse
	citiesOnce sync.Once
}

// QuoteParameters holds the dynamically determined parameters for a booking quote.
type QuoteParameters struct {
	LocationType int
	SpaceID      string
}

func NewWeWork(token string) *WeWork {
	client, err := NewBaseClient()
	if err != nil {
		panic(err)
	}

	// Use the same token for both headers
	client.headers["Authorization"] = []string{"Bearer " + token}
	client.headers["WeWorkAuth"] = []string{"Bearer " + token}

	// Extract UUID from token and add it to headers
	userUUID := extractUUIDFromToken(token)
	if userUUID != "" {
		client.headers["WeWorkUUID"] = []string{userUUID}
	}

	return &WeWork{
		client: client,
	}
}

// Helper function to extract UUID from JWT token
func extractUUIDFromToken(token string) string {
	// Split the token to get the payload
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode the payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	// Parse the JSON
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	// Extract the UUID
	if uuid, ok := claims["https://wework.com/user_uuid"].(string); ok {
		return uuid
	}

	return ""
}

func (w *WeWork) doRequest(method, url string, data any) (*http.Response, error) {
	var body []byte
	var err error

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var errorResp struct {
			ResponseStatus struct {
				Type    string `json:"type"`
				Message string `json:"message"`
				Title   string `json:"title"`
			} `json:"responseStatus"`
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		resp.Body = io.NopCloser(buf)

		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errorResp.ResponseStatus.Type == "error" {
				return nil, fmt.Errorf("API error: %s (%s)", errorResp.ResponseStatus.Message, errorResp.ResponseStatus.Title)
			}
		}

		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp, nil
}

func (w *WeWork) GetLocationsByGeo(city string) (*LocationsByGeoResponse, error) {
	params := url.Values{}
	params.Add("isAuthenticated", "true")
	params.Add("city", city)
	params.Add("isOnDemandUser", "false")
	params.Add("isWeb", "true")

	url := "https://members.wework.com/workplaceone/api/wework-yardi/ondemand/get-locations-by-geo?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LocationsByGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

func (w *WeWork) GetLocationsByGeoCoords(lat, lng float64, delta float64) (*LocationsByGeoResponse, error) {
	if delta <= 0 {
		delta = 0.13
	}
	coords := newGeoCoords(lat, lng, delta)

	params := url.Values{}
	params.Add("isAuthenticated", "true")
	params.Add("isOnDemandUser", "false")
	params.Add("city", "")
	params.Add("userLatitude", strconv.FormatFloat(coords.lat, 'f', 15, 64))
	params.Add("userLongitude", strconv.FormatFloat(coords.lng, 'f', 15, 64))
	params.Add("boundnwLat", strconv.FormatFloat(coords.boundNWLat, 'f', 15, 64))
	params.Add("boundnwLng", strconv.FormatFloat(coords.boundNWLng, 'f', 15, 64))
	params.Add("boundseLat", strconv.FormatFloat(coords.boundSELat, 'f', 15, 64))
	params.Add("boundseLng", strconv.FormatFloat(coords.boundSELng, 'f', 15, 64))

	url := "https://members.wework.com/workplaceone/api/wework-yardi/ondemand/get-locations-by-geo?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LocationsByGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

type geoCoords struct {
	lat        float64
	lng        float64
	boundNWLat float64
	boundNWLng float64
	boundSELat float64
	boundSELng float64
}

func newGeoCoords(lat, lng, delta float64) *geoCoords {
	return &geoCoords{
		lat:        lat,
		lng:        lng,
		boundNWLat: lat + delta,
		boundNWLng: lng - delta,
		boundSELat: lat - delta,
		boundSELng: lng + delta,
	}
}

func (w *WeWork) getAvailableSpacesRequest(t time.Time, locationUUIDs []string, coords *geoCoords) (*SharedWorkspaceResponse, error) {
	params := url.Values{}
	if len(locationUUIDs) > 0 {
		params.Add("locationUUIDs", strings.Join(locationUUIDs, ","))
		// If we have explicit locations, ignore coordinates to avoid constraining results incorrectly.
		coords = nil
	}
	params.Add("closestCity", "")
	if coords != nil {
		params.Add("userLatitude", strconv.FormatFloat(coords.lat, 'f', 7, 64))
		params.Add("userLongitude", strconv.FormatFloat(coords.lng, 'f', 7, 64))

		// Provide a bounding box when no explicit location UUIDs are supplied so the API searches by geo.
		if len(locationUUIDs) == 0 {
			params.Add("boundnwLat", strconv.FormatFloat(coords.boundNWLat, 'f', 7, 64))
			params.Add("boundnwLng", strconv.FormatFloat(coords.boundNWLng, 'f', 7, 64))
			params.Add("boundseLat", strconv.FormatFloat(coords.boundSELat, 'f', 7, 64))
			params.Add("boundseLng", strconv.FormatFloat(coords.boundSELng, 'f', 7, 64))
		} else {
			params.Add("boundnwLat", "")
			params.Add("boundnwLng", "")
			params.Add("boundseLat", "")
			params.Add("boundseLng", "")
		}
	} else {
		params.Add("userLatitude", "")
		params.Add("userLongitude", "")
		params.Add("boundnwLat", "")
		params.Add("boundnwLng", "")
		params.Add("boundseLat", "")
		params.Add("boundseLng", "")
	}
	params.Add("type", "0")
	params.Add("offset", "0")
	params.Add("limit", "50")
	params.Add("roomTypeFilter", "")
	params.Add("date", t.Format("2006-01-02"))
	params.Add("duration", "30")
	params.Add("locationOffset", t.Format("-07:00"))
	params.Add("isWeb", "true")
	params.Add("capacity", "0")
	params.Add("endDate", "")

	url := "https://members.wework.com/workplaceone/api/spaces/get-spaces?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	reader := bytes.NewReader(buf.Bytes())

	var result SharedWorkspaceResponse
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

func (w *WeWork) GetAvailableSpaces(t time.Time, locationUUIDs []string) (*SharedWorkspaceResponse, error) {
	return w.getAvailableSpacesRequest(t, locationUUIDs, nil)
}

func (w *WeWork) GetAvailableSpacesByLatLong(t time.Time, locationUUIDs []string, userLatitude, userLongitude float64) (*SharedWorkspaceResponse, error) {
	return w.getAvailableSpacesRequest(t, locationUUIDs, newGeoCoords(userLatitude, userLongitude, 0.13))
}

func (w *WeWork) GetUpcomingBookings() ([]*Booking, error) {
	url := "https://members.wework.com/workplaceone/api/common-booking/upcoming-bookings"
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// bodyBuf := new(bytes.Buffer)
	// bodyBuf.ReadFrom(resp.Body)

	// reader := bytes.NewReader(bodyBuf.Bytes())

	// buf := new(bytes.Buffer)
	// buf.ReadFrom(reader)
	// fmt.Println(buf.String())

	// reader.Seek(0, 0)
	var result UpcomingBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	for _, booking := range result.Bookings {
		w.adjustBookingTimezone(booking)
	}

	return result.Bookings, nil
}

func (w *WeWork) adjustBookingTimezone(booking *Booking) {
	if booking.Reservable == nil || booking.Reservable.Location == nil {
		return
	}
	loc, err := time.LoadLocation(booking.Reservable.Location.TimeZone)
	if err != nil {
		return
	}

	booking.StartsAt.Time = booking.StartsAt.Time.In(loc)
	booking.EndsAt.Time = booking.EndsAt.Time.In(loc)
}

func (w *WeWork) GetPastBookings() ([]*Booking, error) {
	// Default to past 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)
	return w.GetPastBookingsWithDates(startDate, endDate)
}

func (w *WeWork) GetPastBookingsWithDates(startDate, endDate time.Time) ([]*Booking, error) {
	params := url.Values{}
	params.Add("startDate", startDate.UTC().Format(time.RFC3339))
	params.Add("endDate", endDate.UTC().Format(time.RFC3339))

	url := "https://members.wework.com/workplaceone/api/common-booking/past-bookings?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UpcomingBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	for _, booking := range result.Bookings {
		w.adjustBookingTimezone(booking)
	}

	return result.Bookings, nil
}

func (w *WeWork) GetBootstrap() (*AppBootstrapResponse, error) {
	url := "https://members.wework.com/workplaceone/api/app-bootstrap/bootstrap"

	data := map[string]any{
		"InvalidateCache": false,
		"platform":        1,
		"FeatureFlags": map[string]any{
			"WeGateMemberWebFlags": []string{
				"WG_WEWORK_W_HOMEPAGE_PRINTING",
				"WG_WEWORK_W_MEMWEB_ANNOUNCEMENTS_FROM_CONTENTFUL",
				"WG_WEWORK_W_MEMWEB_BUILDING_GUIDE_UPCOMING_BOOKINGS",
				"WG_WEWORK_W_MEMWEB_ENTERPRISE",
				"WG_WEWORK_W_MEMWEB_EVENTS",
				"WG_WEWORK_W_MEMWEB_SUPPORT_HELP_FAQ",
				"WG_WEWORK_W_MEMWEB_TOP_BANNER_ALL_ACCESS",
				"WG_WEWORK_W_MEMWEB_WEWORK_BRANDING",
				"WG_WEWORK_W_ROOMS_MEDALLIA_SURVEY",
				"WG_WEWORK_W_MEMWEB_WEB_THIRD_PARTY_SPACES",
				"WG_WEWORK_W_MEMWEB_GUEST_POLICY_ENFORCEMENT",
				"WG_WEWORK_W_MEMWEB_PRINT_DRIVER_UPDATE_ROLLOUT",
				"WG_WEWORK_W_MEMWEB_BUILDING_GUIDE_ORGANON_MODULES",
			},
			"WeGateiOSFlags":     []string{},
			"WeGateAndroidFlags": []string{},
		},
		"PermissionRequest": map[string]any{
			"MENAflags": []string{
				"mena_module_building_guide_categories",
				"mena_module_account_manager",
				"mena_module_daily_desks",
				"mena_module_print_hub",
				"mena_module_events",
			},
		},
		"AppVersion":         nil,
		"CurrentAccountUUID": "",
	}

	resp, err := w.doRequest(http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AppBootstrapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

func (w *WeWork) GetUserProfile() (*UserProfileResponse, error) {
	url := "https://members.wework.com/workplaceone/api/wework-yardi/user/get-user-profile"
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

func (w *WeWork) PostBooking(date time.Time, space *Workspace) (*BookingResponse, error) {
	// First get the quote
	quote, err := w.getBookingQuote(date, space)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking quote: %v", err)
	}

	// Then create the booking
	return w.createBooking(date, space, quote)
}

// GetBookingQuote returns the booking quote for a given workspace and date, without creating a booking.
func (w *WeWork) GetBookingQuote(date time.Time, space *Workspace) (*QuoteResponse, error) {
	return w.getBookingQuote(date, space)
}

// getQuoteParameters determines the correct LocationType and SpaceID for a quote.
func getQuoteParameters(space *Workspace) (QuoteParameters, error) {
	if space == nil {
		return QuoteParameters{}, fmt.Errorf("workspace cannot be nil")
	}

	params := QuoteParameters{
		LocationType: space.Location.AccountType,
	}

	// Use InventoryUUID as the primary SpaceID (Tokyo dump shows this is what's actually used)
	// Fall back to UUID if InventoryUUID is empty
	if space.InventoryUUID != "" {
		params.SpaceID = space.InventoryUUID
	} else {
		params.SpaceID = space.UUID
	}

	return params, nil
}

// getBookingSpaceID determines the correct SpaceID for a booking based on LocationType.
// Different WeWork regions/systems use different ID fields for bookings.
func getBookingSpaceID(space *Workspace) string {
	switch space.Location.AccountType {
	case 2:
		// LocationType 2 (e.g., Munich) - use KubeId if available
		if space.Reservable != nil && space.Reservable.KubeId != "" {
			return space.Reservable.KubeId
		}
		// Fallback to inventoryUuid
		if space.InventoryUUID != "" {
			return space.InventoryUUID
		}
		return space.UUID

	case 4:
		// LocationType 4 (e.g., Tokyo) - use inventoryUuid
		if space.InventoryUUID != "" {
			return space.InventoryUUID
		}
		// Fallback to uuid
		return space.UUID

	case 0:
		// LocationType 0 (e.g., Bangkok) - use uuid
		return space.UUID

	default:
		// Unknown LocationType - try inventoryUuid first, then uuid
		if space.InventoryUUID != "" {
			return space.InventoryUUID
		}
		return space.UUID
	}
}

func (w *WeWork) getBookingQuote(date time.Time, space *Workspace) (*QuoteResponse, error) {
	loc, err := time.LoadLocation(space.Location.TimeZone)
	if err != nil {
		return nil, err
	}

	dateInTz := date.In(loc)
	// Parse open and close times (e.g., "08:30" and "20:00")
	openHour, openMin := 8, 30 // Default values
	if len(space.OpenTime) >= 5 {
		fmt.Sscanf(space.OpenTime, "%d:%d", &openHour, &openMin)
	}
	closeHour, closeMin := 20, 0 // Default values
	if len(space.CloseTime) >= 5 {
		fmt.Sscanf(space.CloseTime, "%d:%d", &closeHour, &closeMin)
	}

	// Create start and end times in local timezone
	startLocal := time.Date(dateInTz.Year(), dateInTz.Month(), dateInTz.Day(), openHour, openMin, 0, 0, loc)
	endLocal := time.Date(dateInTz.Year(), dateInTz.Month(), dateInTz.Day(), closeHour, closeMin, 0, 0, loc)

	// Convert to UTC
	startTime := startLocal.UTC().Format("2006-01-02T15:04:05Z")
	endTime := endLocal.UTC().Format("2006-01-02T15:04:05Z")

	quoteURL := "https://members.wework.com/workplaceone/api/common-booking/quote"
	params, err := getQuoteParameters(space)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote parameters: %w", err)
	}

	quoteData := map[string]any{
		"SpaceType":            4,
		"ReservationID":        "",
		"TriggerCalendarEvent": true,
		"Notes":                nil,
		"MailData": map[string]any{
			"dayFormatted":       dateInTz.Format("Monday, January 2nd"),
			"startTimeFormatted": fmt.Sprintf("%s AM", space.OpenTime),
			"endTimeFormatted":   fmt.Sprintf("%s PM", space.CloseTime),
			"floorAddress":       "",
			"locationAddress":    space.Location.Address.Line1,
			"creditsUsed":        "2",
			"Capacity":           "1",
			"TimezoneUsed":       fmt.Sprintf("GMT %s", space.Location.TimezoneOffset),
			"TimezoneIana":       space.Location.TimeZone,
			"startDateTime":      fmt.Sprintf("%s %s", dateInTz.Format("2006-01-02"), space.OpenTime),
			"endDateTime":        fmt.Sprintf("%s %s", dateInTz.Format("2006-01-02"), space.CloseTime),
			"locationName":       space.Location.Name,
			"locationCity":       space.Location.Address.City,
			"locationCountry":    space.Location.Address.Country,
			"locationState":      space.Location.Address.State,
		},
		"LocationType":  params.LocationType,
		"UTCOffset":     space.Location.TimezoneOffset,
		"Currency":      "com.wework.credits",
		"LocationID":    space.Location.UUID,
		"SpaceID":       params.SpaceID,
		"WeWorkSpaceID": space.UUID,
		"StartTime":     startTime,
		"EndTime":       endTime,
	}

	quoteResp, err := w.doRequest(http.MethodPost, quoteURL, quoteData)
	if err != nil {
		return nil, err
	}
	defer quoteResp.Body.Close()

	var quote QuoteResponse
	if err := json.NewDecoder(quoteResp.Body).Decode(&quote); err != nil {
		return nil, fmt.Errorf("failed to decode quote response: %v", err)
	}

	return &quote, nil
}

func (w *WeWork) createBooking(date time.Time, space *Workspace, quote *QuoteResponse) (*BookingResponse, error) {
	loc, err := time.LoadLocation(space.Location.TimeZone)
	if err != nil {
		return nil, err
	}

	dateInTz := date.In(loc)
	// Parse open and close times (e.g., "08:30" and "20:00")
	openHour, openMin := 8, 30 // Default values
	if len(space.OpenTime) >= 5 {
		fmt.Sscanf(space.OpenTime, "%d:%d", &openHour, &openMin)
	}
	closeHour, closeMin := 20, 0 // Default values
	if len(space.CloseTime) >= 5 {
		fmt.Sscanf(space.CloseTime, "%d:%d", &closeHour, &closeMin)
	}

	// Create start and end times in local timezone
	startLocal := time.Date(dateInTz.Year(), dateInTz.Month(), dateInTz.Day(), openHour, openMin, 0, 0, loc)
	endLocal := time.Date(dateInTz.Year(), dateInTz.Month(), dateInTz.Day(), closeHour, closeMin, 0, 0, loc)

	// Convert to UTC
	startTime := startLocal.UTC().Format("2006-01-02T15:04:05Z")
	endTime := endLocal.UTC().Format("2006-01-02T15:04:05Z")

	// Note: Removing the date adjustment logic that was causing "already booked" errors
	// The API should handle far-future dates appropriately
	if daysUntilBooking := time.Until(dateInTz); daysUntilBooking > 30*24*time.Hour {
		// Don't print here - let the caller handle the warning
		// The original date adjustment was causing issues by trying to book past dates
	}

	// Use LocationType-specific logic for booking SpaceID
	bookingSpaceID := getBookingSpaceID(space)

	bookingURL := "https://members.wework.com/workplaceone/api/common-booking/"
	bookingData := map[string]any{
		"ApplicationType":      "WorkplaceOne",
		"PlatformType":         "iOS_APP",
		"SpaceType":            4,
		"ReservationID":        "",
		"TriggerCalendarEvent": true,
		"Notes":                nil,
		"MailData": map[string]any{
			"dayFormatted":       dateInTz.Format("Monday, January 2nd"),
			"startTimeFormatted": fmt.Sprintf("%s AM", space.OpenTime),
			"endTimeFormatted":   fmt.Sprintf("%s PM", space.CloseTime),
			"floorAddress":       "",
			"locationAddress":    space.Location.Address.Line1,
			"creditsUsed":        "0",
			"Capacity":           "1",
			"TimezoneUsed":       fmt.Sprintf("GMT %s", space.Location.TimezoneOffset),
			"TimezoneIana":       space.Location.TimeZone,
			"startDateTime":      fmt.Sprintf("%s %s", dateInTz.Format("2006-01-02"), space.OpenTime),
			"endDateTime":        fmt.Sprintf("%s %s", dateInTz.Format("2006-01-02"), space.CloseTime),
			"locationName":       space.Location.Name,
			"locationCity":       space.Location.Address.City,
			"locationCountry":    space.Location.Address.Country,
			"locationState":      space.Location.Address.State,
		},
		"LocationType":  space.Location.AccountType, // Use dynamic LocationType from accountType
		"UTCOffset":     space.Location.TimezoneOffset,
		"Currency":      "com.wework.credits", // Currency field is required
		"CreditRatio":   quote.GrandTotal.CreditRatio,
		"LocationID":    space.Location.UUID,
		"SpaceID":       bookingSpaceID, // Use LocationType-specific SpaceID
		"WeWorkSpaceID": space.UUID,
		"StartTime":     startTime,
		"EndTime":       endTime,
	}

	bookingResp, err := w.doRequest(http.MethodPost, bookingURL, bookingData)
	if err != nil {
		return nil, err
	}
	defer bookingResp.Body.Close()

	var result BookingResponse
	if err := json.NewDecoder(bookingResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode booking response: %v", err)
	}

	return &result, nil
}

func (w *WeWork) CancelBooking(req CancelBookingRequest, isOnDemand bool, platformType int) (map[string]any, error) {
	params := url.Values{}
	params.Add("isOnDemand", strconv.FormatBool(isOnDemand))
	params.Add("platFormType", strconv.Itoa(platformType))

	cancelURL := "https://members.wework.com/workplaceone/api/common-booking/cancel?" + params.Encode()
	resp, err := w.doRequest(http.MethodPost, cancelURL, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode cancel response: %v", err)
	}

	return result, nil
}

func (w *WeWork) GetCityDetails() ([]*CityDetailsResponse, error) {
	url := "https://members.wework.com/workplaceone/api/wework-yardi/location/get-city-details"
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []*CityDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result, nil
}

func (w *WeWork) GetCities() ([]*CityDetailsResponse, error) {
	var err error
	w.citiesOnce.Do(func() {
		w.cities, err = w.GetCityDetails()
	})
	return w.cities, err
}

func FindCityByFuzzyName(name string, cities []*CityDetailsResponse) ([]*CityDetailsResponse, error) {
	// First, check for exact case-insensitive match
	for _, city := range cities {
		if strings.EqualFold(name, city.Name) {
			return []*CityDetailsResponse{city}, nil
		}
	}

	// If no exact match, perform fuzzy search and return all matches
	var names []string
	for _, city := range cities {
		names = append(names, city.Name)
	}
	matches := fuzzy.Find(name, names)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no city found matching '%s'", name)
	}

	var matchedCities []*CityDetailsResponse
	for _, m := range matches {
		matchedCities = append(matchedCities, cities[m.Index])
	}
	return matchedCities, nil
}

func (w *WeWork) GetLocationFeatures(locationUUID string, amenitiesOnly bool) (*LocationFeaturesResponse, error) {
	params := url.Values{}
	params.Add("locationUUID", locationUUID)
	params.Add("multiple", "false")
	params.Add("amenitiesOnly", fmt.Sprintf("%t", amenitiesOnly))

	url := "https://members.wework.com/workplaceone/api/wework-yardi/location/get-location-features?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LocationFeaturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

// GetSpacesByUUIDs retrieves workspace information for specific location UUIDs
func (w *WeWork) GetSpacesByUUIDs(locationUUIDs []string) (*SharedWorkspaceResponse, error) {
	params := url.Values{}
	params.Add("locationUUIDs", strings.Join(locationUUIDs, ","))
	params.Add("closestCity", "")
	params.Add("userLatitude", "0")
	params.Add("userLongitude", "0")
	params.Add("boundnwLat", "")
	params.Add("boundnwLng", "")
	params.Add("boundseLat", "")
	params.Add("boundseLng", "")
	params.Add("type", "0")
	params.Add("offset", "0")
	params.Add("limit", "500")
	params.Add("roomTypeFilter", "")
	params.Add("date", time.Now().Format("01/02/2006")) // MM/DD/YYYY format - uses current date
	params.Add("duration", "0")
	params.Add("locationOffset", "+00:00")
	params.Add("isWeb", "false")
	params.Add("capacity", "0")
	params.Add("endDate", "")
	params.Add("locationType", "0")
	params.Add("isFromWp", "false")

	url := "https://members.wework.com/workplaceone/api/spaces/get-spaces?" + params.Encode()
	resp, err := w.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SharedWorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}
