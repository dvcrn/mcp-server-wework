package wework

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testWeWorkWithRoundTripper(fn roundTripFunc) *WeWork {
	return &WeWork{
		client: &BaseClient{
			Client: &http.Client{Transport: fn},
			headers: http.Header{
				"Authorization": []string{"Bearer test-token"},
				"WeWorkAuth":    []string{"Bearer test-token"},
			},
		},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

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

func TestGetPrintQueueBuildsPrintHubRequest(t *testing.T) {
	ww := testWeWorkWithRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/workplaceone/api/wework-yardi/print-hub/get-print-queue" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got := req.URL.Query().Get("jobIds"); got != "0" {
			t.Fatalf("expected jobIds=0, got %q", got)
		}
		if got := req.Header.Get("fe-pg"); got != "/workplaceone/content2/print" {
			t.Fatalf("unexpected fe-pg header: %q", got)
		}
		if got := req.Header.Get("Request-Source"); got != "MemberWeb/WorkplaceOne/Prod" {
			t.Fatalf("unexpected request source: %q", got)
		}
		return jsonResponse(http.StatusOK, `{"content":[],"page":{"totalElements":0}}`), nil
	})

	result, err := ww.GetPrintQueue("")
	if err != nil {
		t.Fatalf("GetPrintQueue returned error: %v", err)
	}
	if result.Page.TotalElements != 0 {
		t.Fatalf("unexpected total elements: %d", result.Page.TotalElements)
	}
	if len(result.Content) != 0 {
		t.Fatalf("expected empty content, got %d rows", len(result.Content))
	}
}

func TestAddToPrintQueueBuildsMultipartRequest(t *testing.T) {
	ww := testWeWorkWithRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/workplaceone/api/wework-yardi/print-hub/add-to-print-queue" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("fe-pg"); got != "/workplaceone/content2/print" {
			t.Fatalf("unexpected fe-pg header: %q", got)
		}

		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("failed to parse content type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("expected multipart/form-data, got %s", mediaType)
		}

		reader := multipart.NewReader(req.Body, params["boundary"])
		form, err := reader.ReadForm(1024 * 1024)
		if err != nil {
			t.Fatalf("failed to read multipart form: %v", err)
		}

		expectedFields := map[string]string{
			"copies":               "2",
			"forceMediaSize":       "iso_a4_210x297mm",
			"orientationRequested": "landscape",
			"printColorMode":       "color",
			"sides":                "two-sided-long-edge",
			"jobName":              "test-job.pdf",
		}
		for key, expected := range expectedFields {
			if got := form.Value[key]; len(got) != 1 || got[0] != expected {
				t.Fatalf("field %s = %v, expected %q", key, got, expected)
			}
		}
		files := form.File["file"]
		if len(files) != 1 {
			t.Fatalf("expected one file, got %d", len(files))
		}
		if files[0].Filename != "test.pdf" {
			t.Fatalf("unexpected file name: %s", files[0].Filename)
		}
		if files[0].Header.Get("Content-Type") != "application/pdf" {
			t.Fatalf("unexpected file content type: %s", files[0].Header.Get("Content-Type"))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatalf("failed to open uploaded file part: %v", err)
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file part: %v", err)
		}
		if string(content) != "%PDF-1.4 test" {
			t.Fatalf("unexpected file content: %q", string(content))
		}

		return jsonResponse(http.StatusOK, `{"id":"job-123","status":"RECEIVED","jobName":"test-job.pdf","copies":2,"printColorMode":"color","sides":"two-sided-long-edge"}`), nil
	})

	job, err := ww.AddToPrintQueue(AddPrintJobRequest{
		Copies:               2,
		ForceMediaSize:       "iso_a4_210x297mm",
		OrientationRequested: "landscape",
		PrintColorMode:       "color",
		Sides:                "two-sided-long-edge",
		JobName:              "test-job.pdf",
		FileName:             "test.pdf",
		FileContentType:      "application/pdf",
		FileBytes:            []byte("%PDF-1.4 test"),
	})
	if err != nil {
		t.Fatalf("AddToPrintQueue returned error: %v", err)
	}

	encoded, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("failed to marshal job: %v", err)
	}
	if !strings.Contains(string(encoded), `"id":"job-123"`) {
		t.Fatalf("unexpected job response: %s", encoded)
	}
}
