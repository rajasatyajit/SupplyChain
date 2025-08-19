package geocoder

import (
	"testing"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

func TestGeocoder_Geocode(t *testing.T) {
	geocoder := New()

	tests := []struct {
		name             string
		alert            models.Alert
		expectedLocation string
		expectedCountry  string
		expectedRegion   string
	}{
		{
			name: "Port location extraction",
			alert: models.Alert{
				Title:   "Strike at Port of Los Angeles",
				Summary: "Major disruption at the port facility",
			},
			expectedLocation: "Port of Los Angeles",
			expectedCountry:  "",
			expectedRegion:   "",
		},
		{
			name: "City and state extraction",
			alert: models.Alert{
				Title:   "Traffic delays in Seattle, WA",
				Summary: "Heavy congestion reported",
			},
			expectedLocation: "Seattle, WA",
			expectedCountry:  "",
			expectedRegion:   "",
		},
		{
			name: "No location found",
			alert: models.Alert{
				Title:   "General supply chain update",
				Summary: "Overall market conditions",
			},
			expectedLocation: "",
			expectedCountry:  "",
			expectedRegion:   "",
		},
		{
			name: "Multiple locations - first match",
			alert: models.Alert{
				Title:   "Issues at Port of Miami and Port of Tampa",
				Summary: "Multiple facilities affected",
			},
			expectedLocation: "Port of Miami",
			expectedCountry:  "",
			expectedRegion:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := geocoder.Geocode(&tt.alert)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.alert.Location != tt.expectedLocation {
				t.Errorf("Expected location %s, got %s", tt.expectedLocation, tt.alert.Location)
			}

			// Latitude and longitude should remain 0 in this implementation
			if tt.alert.Latitude != 0.0 {
				t.Errorf("Expected latitude 0.0, got %f", tt.alert.Latitude)
			}

			if tt.alert.Longitude != 0.0 {
				t.Errorf("Expected longitude 0.0, got %f", tt.alert.Longitude)
			}
		})
	}
}

func TestGeocoder_ExtractRegionAndCountry(t *testing.T) {
	geocoder := New()

	tests := []struct {
		name            string
		location        string
		expectedCountry string
		expectedRegion  string
	}{
		{
			name:            "US location",
			location:        "Port of Los Angeles, US",
			expectedCountry: "United States",
			expectedRegion:  "North America",
		},
		{
			name:            "UK location",
			location:        "London, UK",
			expectedCountry: "United Kingdom",
			expectedRegion:  "Europe",
		},
		{
			name:            "Germany location",
			location:        "Hamburg, DE",
			expectedCountry: "Germany",
			expectedRegion:  "Europe",
		},
		{
			name:            "Unknown location",
			location:        "Unknown City",
			expectedCountry: "",
			expectedRegion:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &models.Alert{}
			geocoder.extractRegionAndCountry(alert, tt.location)

			if alert.Country != tt.expectedCountry {
				t.Errorf("Expected country %s, got %s", tt.expectedCountry, alert.Country)
			}

			if alert.Region != tt.expectedRegion {
				t.Errorf("Expected region %s, got %s", tt.expectedRegion, alert.Region)
			}
		})
	}
}

func TestNew(t *testing.T) {
	geocoder := New()

	if geocoder == nil {
		t.Error("Expected geocoder instance, got nil")
	}

	if geocoder.cityRegex == nil {
		t.Error("Expected city regex to be initialized")
	}
}
