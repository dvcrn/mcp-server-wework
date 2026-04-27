package wework

import (
	"strings"
	"testing"
)

func TestGetQuoteParameters(t *testing.T) {
	// Test case for Munich - uses inventoryUuid when available
	munichWorkspace := &Workspace{
		UUID:          "6ac970ee-972c-11e8-b488-0ac77f0f6524",
		InventoryUUID: "munich-inventory-uuid",
		Location: Location{
			AccountType: 2,
		},
		Reservable: &WorkspaceReservable{
			KubeId: "131834",
		},
	}

	// Test case for Bangkok - falls back to UUID when no inventoryUuid
	bangkokWorkspace := &Workspace{
		UUID:          "c61971d2-624d-11e9-a390-0e1e2abc3cd0",
		InventoryUUID: "", // No inventory UUID
		Location: Location{
			AccountType: 0,
		},
		Reservable: nil, // No "reservable" object in the response
	}

	// Test case for Tokyo - based on actual dump showing inventoryUuid is used
	tokyoWorkspace := &Workspace{
		UUID:          "eb08c128-e25f-11e8-9de1-0ac77f0f6524",
		InventoryUUID: "52043b70-0bf7-4707-8a6a-b7982dff823b", // From actual Tokyo dump
		Location: Location{
			AccountType: 4,
		},
		Reservable: &WorkspaceReservable{
			KubeId: "6147",
		},
	}

	testCases := []struct {
		name           string
		workspace      *Workspace
		expectedParams QuoteParameters
		expectError    bool
	}{
		{
			name:      "Munich - Uses inventoryUuid",
			workspace: munichWorkspace,
			expectedParams: QuoteParameters{
				LocationType: 2,
				SpaceID:      "munich-inventory-uuid",
			},
			expectError: false,
		},
		{
			name:      "Bangkok - Falls back to UUID",
			workspace: bangkokWorkspace,
			expectedParams: QuoteParameters{
				LocationType: 0,
				SpaceID:      "c61971d2-624d-11e9-a390-0e1e2abc3cd0",
			},
			expectError: false,
		},
		{
			name:      "Tokyo - Uses inventoryUuid not KubeId",
			workspace: tokyoWorkspace,
			expectedParams: QuoteParameters{
				LocationType: 4,
				SpaceID:      "52043b70-0bf7-4707-8a6a-b7982dff823b",
			},
			expectError: false,
		},
		{
			name:        "Nil Workspace",
			workspace:   nil,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := getQuoteParameters(tc.workspace)

			if (err != nil) != tc.expectError {
				t.Fatalf("getQuoteParameters() error = %v, expectError %v", err, tc.expectError)
			}

			if !tc.expectError {
				if params.LocationType != tc.expectedParams.LocationType {
					t.Errorf("Expected LocationType to be %d, but got %d", tc.expectedParams.LocationType, params.LocationType)
				}
				if params.SpaceID != tc.expectedParams.SpaceID {
					t.Errorf("Expected SpaceID to be '%s', but got '%s'", tc.expectedParams.SpaceID, params.SpaceID)
				}
			}
		})
	}
}

func TestFindCityByFuzzyName(t *testing.T) {
	cities := []*CityDetailsResponse{
		{Name: "Tokyo"},
		{Name: "New York"},
		{Name: "London"},
		{Name: "Paris"},
		{Name: "Berlin"},
	}

	tests := []struct {
		name           string
		searchName     string
		expectedCities []string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "exact match",
			searchName:     "Tokyo",
			expectedCities: []string{"Tokyo"},
			expectError:    false,
		},
		{
			name:           "fuzzy match single result",
			searchName:     "York",
			expectedCities: []string{"New York"},
			expectError:    false,
		},
		{
			name:           "fuzzy match multiple results",
			searchName:     "o",
			expectedCities: []string{"Tokyo", "New York", "London"},
			expectError:    false,
		},
		{
			name:          "no match",
			searchName:    "Nonexistent",
			expectError:   true,
			errorContains: "no city found",
		},
		{
			name:           "case insensitive exact match",
			searchName:     "tokyo",
			expectedCities: []string{"Tokyo"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchedCities, err := FindCityByFuzzyName(tt.searchName, cities)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				actualNames := make([]string, len(matchedCities))
				for i, city := range matchedCities {
					actualNames[i] = city.Name
				}
				if len(actualNames) != len(tt.expectedCities) {
					t.Errorf("expected %d cities, got %d: %v", len(tt.expectedCities), len(actualNames), actualNames)
				}
				for _, expected := range tt.expectedCities {
					found := false
					for _, actual := range actualNames {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected city %s not found in %v", expected, actualNames)
					}
				}
			}
		})
	}
}
