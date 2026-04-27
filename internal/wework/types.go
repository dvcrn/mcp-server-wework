package wework

import (
	"strings"
	"time"
)

// CustomTime is a wrapper around time.Time to handle custom date formats
type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" || s == "0001-01-01T00:00:00Z" || s == "01/01/0001 00:00:00" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(time.RFC3339, s)
	return
}

type SharedWorkspaceResponse struct {
	Limit    int `json:"limit"`
	Offset   int `json:"offset"`
	Response struct {
		Workspaces []Workspace `json:"workspaces"`
	} `json:"getSharedWorkspaces"`
}

type LocationsByGeoResponse struct {
	LocationsByGeo []GeoLocation `json:"locationsByGeo"`
}

type BookingResponse struct {
	BookingStatus string   `json:"BookingStatus"`
	Errors        []string `json:"Errors"`
	ReservationID string   `json:"ReservationID"`
}

type CancelBookingRequest struct {
	BookingID           string         `json:"bookingId"`
	BookingLocationType int            `json:"bookingLocationType"`
	ReservableID        string         `json:"reservableId,omitempty"`
	StartTime           string         `json:"startTime,omitempty"`
	EndTime             string         `json:"endTime,omitempty"`
	CreditsUsed         string         `json:"creditsUsed,omitempty"`
	LocationID          string         `json:"locationId,omitempty"`
	MailParams          CancelMailData `json:"mailParams,omitempty"`
	BookingType         int            `json:"bookingType,omitempty"`
	ReservationID       string         `json:"reservationId,omitempty"`
}

type CancelMailData struct {
	WorkspaceType      string `json:"workspaceType,omitempty"`
	DayFormatted       string `json:"dayFormatted,omitempty"`
	StartTimeFormatted string `json:"startTimeFormatted,omitempty"`
	EndTimeFormatted   string `json:"endTimeFormatted,omitempty"`
	FloorAddress       string `json:"floorAddress,omitempty"`
	LocationAddress    string `json:"locationAddress,omitempty"`
}

type GeoLocation struct {
	UUID                 string  `json:"uuid"`
	Name                 string  `json:"name"`
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	Address              Address `json:"address"`
	TimeZone             string  `json:"timeZone"`
	Distance             float64 `json:"distance"`
	BrandName            string  `json:"brandName"`
	HasThirdPartyDisplay bool    `json:"hasThirdPartyDisplay"`
	Image                string  `json:"image"`
	IsMigrated           bool    `json:"isMigrated"`
}

type UpcomingBookingsResponse struct {
	Bookings []*Booking `json:"WeWorkBookings"`
}

type Booking struct {
	UUID                         string           `json:"uuid"`
	StartsAt                     CustomTime       `json:"startsAt"`
	EndsAt                       CustomTime       `json:"endsAt"`
	TimeZone                     string           `json:"timezone"`
	CreditOrder                  *CreditOrder     `json:"creditOrder"`
	Reservable                   *SharedWorkspace `json:"reservable"`
	IsAttendee                   bool             `json:"isAttendee"`
	ModificationDeadline         CustomTime       `json:"modificationDeadline"`
	Order                        Order            `json:"order"`
	IsMultidayBooking            bool             `json:"isMultidayBooking"`
	KubeSameDayCancelPolicy      bool             `json:"kubeSameDayCancelPolicy"`
	IsFromKube                   bool             `json:"isFromKube"`
	KubeBookingExternalReference string           `json:"kubeBookingExternalReference"`
	CwmBookingReferenceID        int              `json:"cwmBookingReferenceId"`
	IsFromCwm                    bool             `json:"isFromCwm"`
	IsBookingConfirmationPending bool             `json:"isBookingConfirmationPending"`
	IsBookingApprovalOn          bool             `json:"IsBookingApprovalOn"`
	SameDayCancelPolicy          bool             `json:"sameDayCancelPolicy"`
	// KubeCreatedOnDate            *time.Time      `json:"kubeCreatedOnDate,omitempty"`
	// KubeModifiedOnDate           *time.Time      `json:"kubeModifiedOnDate,omitempty"`
	// KubeStartDate                *time.Time      `json:"kubeStartDate,omitempty"`
}

type CreditOrder struct {
	Price string `json:"price"`
}

type SharedWorkspace struct {
	UUID       string                   `json:"uuid"`
	Name       string                   `json:"name"`
	FloorName  string                   `json:"floorName"`
	Capacity   int                      `json:"capacity"`
	TypeName   string                   `json:"__typename"`
	Location   *SharedWorkspaceLocation `json:"location"`
	ImageURL   string                   `json:"imageUrl"`
	CwmSpaceID int                      `json:"cwmSpaceId"`
}

type SharedWorkspaceLocation struct {
	KubePropertyID       int     `json:"kubePropertyID"`
	CwmPropertyID        int     `json:"cwmPropertyID"`
	SourceType           int     `json:"sourceType"`
	UUID                 string  `json:"uuid"`
	Name                 string  `json:"name"`
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	Address              Address `json:"address"`
	TimeZone             string  `json:"timeZone"`
	TimeZoneIdentifier   string  `json:"timeZoneIdentifier"`
	Distance             float64 `json:"distance"`
	HasThirdPartyDisplay bool    `json:"hasThirdPartyDisplay"`
	IsMigrated           bool    `json:"isMigrated"`
}

type Order struct {
	PaymentProfileUUID string `json:"paymentProfileUuid"`
	SubTotal           Amount `json:"subTotal"`
	Adjustments        []any  `json:"adjustments"`
	Taxes              []any  `json:"taxes"`
	GrandTotal         Amount `json:"grandTotal"`
}

type Amount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Address struct {
	Line1   string `json:"line1"`
	Line2   string `json:"line2"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
	Zip     string `json:"zip"`
}

type Workspace struct {
	UUID                 string               `json:"uuid"`
	InventoryUUID        string               `json:"inventoryUuid"`
	ImageURL             string               `json:"imageURL"`
	HeaderImageURL       string               `json:"headerImageUrl"`
	Capacity             int                  `json:"capacity"`
	Credits              int                  `json:"credits"`
	Location             Location             `json:"location"`
	OpenTime             string               `json:"openTime"`
	CloseTime            string               `json:"closeTime"`
	CancellationPolicy   string               `json:"cancellationPolicy"`
	OperatingHours       []*OperatingHours    `json:"operatingHours"`
	ProductPrice         *ProductPrice        `json:"productPrice"`
	Seat                 Seat                 `json:"seat"`
	SeatsAvailable       int                  `json:"seatsAvailable"`
	Order                any                  `json:"order"`
	IsVASTCoworking      bool                 `json:"isVASTCoworking"`
	IsAffiliateCoworking bool                 `json:"isAffiliateCoworking"`
	IsFranchiseCoworking bool                 `json:"isFranchiseCoworking"`
	IsHybridSpace        bool                 `json:"isHybridSpace"`
	Reservable           *WorkspaceReservable `json:"reservable"`
}

type WorkspaceReservable struct {
	Capacity      int    `json:"capacity"`
	KubeId        string `json:"KubeId"`
	CwmSpaceId    int    `json:"cwmSpaceId"`
	CwmSpaceCount int    `json:"cwmSpaceCount"`
}

type Location struct {
	Description                string             `json:"description"`
	SupportEmail               string             `json:"supportEmail"`
	PhoneNormalized            string             `json:"phoneNormalized"`
	Currency                   string             `json:"currency"`
	PrimaryTeamMember          TeamMember         `json:"primaryTeamMember"`
	Amenities                  []Amenity          `json:"amenities"`
	Details                    Details            `json:"details"`
	TransitInfo                TransitInfo        `json:"transitInfo"`
	MemberEntranceInstructions string             `json:"memberEntranceInstructions"`
	ParkingInstructions        string             `json:"parkingInstructions"`
	CommunityBarFloor          *CommunityBarFloor `json:"communityBarFloor,omitempty"`
	TimezoneOffset             string             `json:"timezoneOffset"`
	TimeZoneIdentifier         string             `json:"timeZoneIdentifier"`
	TimeZoneWinID              string             `json:"timeZoneWinId"`
	UUID                       string             `json:"uuid"`
	Name                       string             `json:"name"`
	Latitude                   float64            `json:"latitude"`
	Longitude                  float64            `json:"longitude"`
	Address                    Address            `json:"address"`
	TimeZone                   string             `json:"timeZone"`
	Distance                   float64            `json:"distance"`
	HasThirdPartyDisplay       bool               `json:"hasThirdPartyDisplay"`
	IsMigrated                 bool               `json:"isMigrated"`
	AccountType                int                `json:"accountType"`
	Images                     []LocationImage    `json:"images,omitempty"`
}

type TeamMember struct {
	Name          string `json:"name"`
	BusinessTitle string `json:"businessTitle"`
	ImageURL      string `json:"imageUrl"`
}

type Amenity struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Highlight bool   `json:"highlight"`
}

type Details struct {
	HasExtendedHours bool `json:"hasExtendedHours"`
}

type TransitInfo struct {
	Bike    string `json:"bike"`
	Bus     string `json:"bus"`
	Ferry   string `json:"ferry"`
	Freeway string `json:"freeway"`
	Metro   string `json:"metro"`
	Parking string `json:"parking"`
}

type OperatingHours struct {
	DayOfWeek int    `json:"dayOfWeek"`
	Day       string `json:"day"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	IsClosed  bool   `json:"isClosed"`
}

type ProductPrice struct {
	UUID                 string                 `json:"uuid"`
	ProductUUID          string                 `json:"productUuid"`
	Price                Price                  `json:"price"`
	RateUnit             int                    `json:"rateUnit"`
	HalfHourCreditPrices []*HalfHourCreditPrice `json:"halfHourCreditPrices"`
}

type Price struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type HalfHourCreditPrice struct {
	Offset int     `json:"offset"`
	Amount float64 `json:"amount"`
}

type Seat struct {
	Total     int `json:"total"`
	Available int `json:"available"`
}

type QuoteResponse struct {
	UUID          string     `json:"uuid"`
	QuoteStatus   int        `json:"quoteStatus"`
	StatusDetails []any      `json:"statusDetails"`
	GrandTotal    Money      `json:"grandTotal"`
	SubTotal      Money      `json:"subTotal"`
	Taxes         []any      `json:"taxes"`
	LineItems     []LineItem `json:"lineItems"`
	Adjustments   []any      `json:"adjustments"`
}

type Money struct {
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	CreditRatio float64 `json:"creditRatio,omitempty"`
}

type LineItem struct{}

type AppBootstrapRequest struct {
	IsCAKube         bool   `json:"isCAKube"`
	IsKube           bool   `json:"isKube"`
	RequestSource    string `json:"requestSource"`
	WeWorkMemberType string `json:"weWorkMemberType"`
	WeWorkUUID       string `json:"weWorkUUID"`
}

type AppBootstrapResponse struct {
	MenuSecurityData struct {
		IsPasswordChangeEnforcing bool `json:"isPasswordChangeEnforcing"`
		MenuData                  []struct {
			Id               int    `json:"Id"`
			MenuItemName     string `json:"MenuItemName"`
			MenuCaption      string `json:"MenuCaption"`
			MenuLinkURL      string `json:"MenuLinkURL"`
			Parent           int    `json:"Parent"`
			Order            int    `json:"Order"`
			HasSubMenu       bool   `json:"HasSubMenu"`
			OpenInNewWindow  bool   `json:"OpenInNewWindow"`
			MenuSet          string `json:"MenuSet"`
			NavCssClass      string `json:"NavCssClass"`
			ActiveClass      string `json:"ActiveClass"`
			MenuIconImage    string `json:"MenuIconImage"`
			MenuContentClass int    `json:"MenuContentClass"`
			RouteParams      string `json:"RouteParams"`
			MenuSection      int    `json:"MenuSection"`
			IsToBeRemoved    bool   `json:"IsToBeRemoved"`
		} `json:"menuData"`
		Cacheresponse bool   `json:"Cacheresponse"`
		AdminRole     string `json:"adminRole"`
	} `json:"menuSecurityData"`
	PageSentryData struct {
		AllowedItems []struct {
			Identifier  int    `json:"identifier"`
			URLFragment string `json:"urlFragment"`
		} `json:"allowedItems"`
	} `json:"pageSentryData"`
	WeworkUserProfileData struct {
		ProfileData struct {
			WeWorkCompanyLicenseType int `json:"WeWorkCompanyLicenseType"`
			WeWorkUserData           struct {
				WeWorkMembershipUUID            string  `json:"WeWorkMembershipUUID"`
				WeWorkProductUUID               string  `json:"WeWorkProductUUID"`
				WeWorkPreferredMembershipUUID   string  `json:"WeWorkPreferredMembershipUUID"`
				WeWorkUserUUID                  string  `json:"WeWorkUserUUID"`
				WeWorkCompanyUUID               string  `json:"WeWorkCompanyUUID"`
				WeWorkChargableAccountUUID      string  `json:"WeWorkChargableAccountUUID"`
				WeWorkCompanyMigrationStatus    string  `json:"WeWorkCompanyMigrationStatus"`
				WeWorkUserEmail                 string  `json:"WeWorkUserEmail"`
				WeWorkMembershipType            string  `json:"WeWorkMembershipType"`
				WeWorkMembershipName            string  `json:"WeWorkMembershipName"`
				WeWorkUserName                  string  `json:"WeWorkUserName"`
				WeWorkUserPhone                 string  `json:"WeWorkUserPhone"`
				WeWorkUserAvatar                string  `json:"WeWorkUserAvatar"`
				WeWorkUserLanguagePreference    string  `json:"WeWorkUserLanguagePreference"`
				WeWorkUserHomeLocationUUID      string  `json:"WeWorkUserHomeLocationUUID"`
				WeWorkUserHomeLocationName      string  `json:"WeWorkUserHomeLocationName"`
				WeWorkUserHomeLocationCity      string  `json:"WeWorkUserHomeLocationCity"`
				WeWorkUserHomeLocationLatitude  float64 `json:"WeWorkUserHomeLocationLatitude"`
				WeWorkUserHomeLocationLongitude float64 `json:"WeWorkUserHomeLocationLongitude"`
				WeWorkUserHomeLocationMigrated  bool    `json:"WeWorkUserHomeLocationMigrated"`
				WeWorkUserPreferredCurrency     string  `json:"WeWorkUserPreferredCurrency"`
				NoActiveMemberships             bool    `json:"NoActiveMemberships"`
				IsKubeMigratedAccount           bool    `json:"IsKubeMigratedAccount"`
				WeWorkUserThemePreference       int     `json:"WeWorkUserThemePreference"`
			} `json:"WeWorkUserData"`
			WeWorkCompanyList []struct {
				UserUUID                       string `json:"UserUUID"`
				CompanyUUID                    string `json:"CompanyUUID"`
				CompanyName                    string `json:"CompanyName"`
				CompanyLicenseType             int    `json:"CompanyLicenseType"`
				PreferredMembershipUUID        string `json:"PreferredMembershipUUID"`
				PreferredMembershipName        string `json:"PreferredMembershipName"`
				PreferredMembershipType        string `json:"PreferredMembershipType"`
				PreferredMembershipProductUUID string `json:"PreferredMembershipProductUUID"`
				IsMigratedToKUBE               bool   `json:"IsMigratedToKUBE"`
				CompanyMigrationStatus         string `json:"CompanyMigrationStatus"`
				KUBECompanyUUID                string `json:"KUBECompanyUUID"`
			} `json:"WeWorkCompanyList"`
			WeWorkMembershipsList []struct {
				UUID           string `json:"uuid"`
				AccountUUID    string `json:"accountUuid"`
				MembershipType string `json:"membershipType"`
				UserUUID       string `json:"userUuid"`
				ProductName    string `json:"productName"`
				ProductUUID    string `json:"productUuid"`
				ActivatedOn    string `json:"activatedOn"`
				StartedOn      string `json:"startedOn"`
				IsMigrated     bool   `json:"isMigrated"`
				PriorityOrder  int    `json:"priorityOrder"`
			} `json:"WeWorkMembershipsList"`
			UserOnboardingStatus bool   `json:"UserOnboardingStatus"`
			DebugModeEnabled     bool   `json:"DebugModeEnabled"`
			IsUserWorkplaceAdmin bool   `json:"IsUserWorkplaceAdmin"`
			AccountManagerLink   string `json:"AccountManagerLink"`
		} `json:"profileData"`
	} `json:"weworkUserProfileData"`
	WeGateData struct {
		WeGateFlags struct {
			WeGateMemberWebFlags struct {
				Meta struct{} `json:"meta"`
				Data []any    `json:"data"`
			} `json:"WeGateMemberWebFlags"`
		} `json:"weGateFlags"`
	} `json:"weGateData"`
	WorkplaceExperienceStatus bool `json:"WorkplaceExperienceStatus"`
	VastExperienceStatus      bool `json:"VastExperienceStatus"`
	GlobalSettings            struct {
		AllowAffiliateBookingsInMemberWeb bool `json:"AllowAffiliateBookingsInMemberWeb"`
	} `json:"GlobalSettings"`
}

type UserProfileResponse struct {
	UUID               string       `json:"uuid"`
	Phone              string       `json:"phone"`
	Email              string       `json:"email"`
	AvatarURL          string       `json:"avatarUrl"`
	CoverURL           string       `json:"coverUrl"`
	Name               string       `json:"name"`
	IsWework           bool         `json:"isWework"`
	IsAdmin            bool         `json:"isAdmin"`
	IsAdminOrCM        bool         `json:"isAdminOrCM"`
	Active             bool         `json:"active"`
	AccountState       int          `json:"accountState"`
	LanguagePreference string       `json:"languagePreference"`
	HomeLocation       HomeLocation `json:"homeLocation"`
	Companies          []Company    `json:"companies"`
	RegistrationInfo   Registration `json:"registrationInfo"`
}

type HomeLocation struct {
	UUID            string  `json:"uuid"`
	Name            string  `json:"name"`
	SupportEmail    string  `json:"supportEmail"`
	PhoneNormalized string  `json:"phoneNormalized"`
	Currency        string  `json:"currency"`
	TimeZone        string  `json:"timeZone"`
	Locale          string  `json:"locale"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Address         Address `json:"address"`
	Market          Market  `json:"market"`
	IsMigrated      bool    `json:"isMigrated"`
}

type Market struct {
	UUID string `json:"uuid"`
}

type Company struct {
	UUID                        string               `json:"uuid"`
	Name                        string               `json:"name"`
	IsMigrated                  bool                 `json:"isMigrated"`
	MigrationStatus             string               `json:"migrationStatus"`
	PreferredMembershipNullable *PreferredMembership `json:"preferredMembershipNullable"`
}

type PreferredMembership struct {
	UUID           string `json:"uuid"`
	AccountUUID    string `json:"accountUuid"`
	MembershipType string `json:"membershipType"`
	UserUUID       string `json:"userUuid"`
	ProductUUID    string `json:"productUuid"`
	ProductName    string `json:"productName"`
	IsMigrated     bool   `json:"isMigrated"`
}

type Registration struct {
	Country Country `json:"country"`
}

type Country struct {
	Name   string `json:"name"`
	Alpha2 string `json:"alpha2"`
	Alpha3 string `json:"alpha3"`
	Code   int    `json:"code"`
	Region int    `json:"region"`
}

type CityDetailsResponse struct {
	Name                 string         `json:"name"`
	MarketGeo            MarketGeo      `json:"marketgeo"`
	CountryGeo           CountryGeo     `json:"countrygeo"`
	NearbyLocation       NearbyLocation `json:"nearby_location"`
	NearbyLocationsCount int            `json:"nearby_locations_count"`
}

type LocationFeaturesResponse struct {
	Locations []LocationFeatures `json:"locations"`
}

type LocationFeatures struct {
	UUID                       string            `json:"uuid"`
	Name                       string            `json:"name"`
	Description                string            `json:"description"`
	SupportEmail               string            `json:"supportEmail"`
	Phone                      string            `json:"phone"`
	PhoneNormalized            string            `json:"phoneNormalized"`
	Currency                   string            `json:"currency"`
	TimeZone                   string            `json:"timeZone"`
	Latitude                   float64           `json:"latitude"`
	Longitude                  float64           `json:"longitude"`
	Address                    Address           `json:"address"`
	EntranceAddress            Address           `json:"entranceAddress"`
	PrimaryTeamMember          TeamMember        `json:"primaryTeamMember"`
	Images                     []LocationImage   `json:"images"`
	Amenities                  []Amenity         `json:"amenities"`
	Details                    LocationDetails   `json:"details"`
	TransitInfo                TransitInfo       `json:"transitInfo"`
	MemberEntranceInstructions string            `json:"memberEntranceInstructions"`
	TourInstructions           string            `json:"tourInstructions"`
	ParkingInstructions        string            `json:"parkingInstructions"`
	CommunityBarFloor          CommunityBarFloor `json:"communityBarFloor"`
	BrandName                  string            `json:"brandName"`
	HasThirdPartyDisplay       bool              `json:"hasThirdPartyDisplay"`
	IsMigrated                 bool              `json:"isMigrated"`
}

type LocationImage struct {
	UUID     string `json:"uuid"`
	Caption  string `json:"caption"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

type LocationDetails struct {
	HasExtendedHours bool              `json:"hasExtendedHours"`
	OperatingHours   []OperatingDetail `json:"operatingHours"`
	CommunityBar     CommunityBar      `json:"communityBar"`
}

type OperatingDetail struct {
	DayOfWeek string `json:"dayOfWeek"`
	TimeOpen  string `json:"timeOpen,omitempty"`
	TimeClose string `json:"timeClose,omitempty"`
}

type CommunityBar struct {
	FloorNumber int `json:"floorNumber"`
}

type CommunityBarFloor struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type MarketGeo struct {
	ID               string  `json:"id"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Name             string  `json:"name"`
	NameAbbreviation string  `json:"name_abbreviation"`
}

type CountryGeo struct {
	ID               string  `json:"id"`
	ISO              string  `json:"iso"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Name             string  `json:"name"`
	NameAbbreviation string  `json:"name_abbreviation"`
}

type NearbyLocation struct {
	UUID                          string  `json:"uuid"`
	DefaultName                   string  `json:"default_name"`
	DefaultCountry                string  `json:"default_country"`
	NameOverride                  string  `json:"name_override"`
	Description                   string  `json:"description"`
	Latitude                      float64 `json:"latitude"`
	Longitude                     float64 `json:"longitude"`
	DefaultLocale                 string  `json:"default_locale"`
	TimeZone                      string  `json:"time_zone"`
	DefaultCurrency               string  `json:"default_currency"`
	Line1                         string  `json:"line1"`
	Line2                         string  `json:"line2"`
	City                          string  `json:"city"`
	State                         string  `json:"state"`
	Zip                           string  `json:"zip"`
	Address                       string  `json:"address"`
	IsOpen                        bool    `json:"is_open"`
	Slug                          string  `json:"slug"`
	Phone                         string  `json:"phone"`
	PhoneNormalized               string  `json:"phone_normalized"`
	Code                          string  `json:"code"`
	EntranceInstructions          string  `json:"entrance_instructions"`
	EntranceInstructionsLocalized string  `json:"entrance_instructions_localized"`
	DefaultCountryCode            string  `json:"default_country_code"`
	NormalizedDefaultCurrency     string  `json:"normalized_default_currency"`
	IsFavorite                    bool    `json:"is_favorite"`
	IsPublished                   bool    `json:"is_published"`
	AccessType                    string  `json:"access_type"`
	OnDemandEnabled               bool    `json:"on_demand_enabled"`
	BrandName                     string  `json:"brand_name"`
}
