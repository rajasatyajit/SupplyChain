package geocoder

import (
	"regexp"
	"strings"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// Geocoder provides geolocation functionality for alerts
type Geocoder struct {
	cityRegex *regexp.Regexp
}

// New creates a new geocoder instance
func New() *Geocoder {
	return &Geocoder{
		// Match "Port of X Y" (case-insensitive for the phrase 'Port of') or "City, ST"
		cityRegex: regexp.MustCompile(`\b((?i:port of)\s+[A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)?|[A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)*,\s*[A-Z]{2})\b`),
	}
}

// Geocode extracts location information from an alert
func (g *Geocoder) Geocode(alert *models.Alert) error {
	text := alert.Title + " " + alert.Summary
	
	// Extract location using regex
	if loc := g.cityRegex.FindString(text); loc != "" {
		alert.Location = loc
		
		// In a production system, you would:
		// 1. Use a proper geocoding service (Google Maps, OpenStreetMap, etc.)
		// 2. Cache results to avoid repeated API calls
		// 3. Handle rate limiting
		// 4. Set actual latitude/longitude coordinates
		
		// For now, we'll leave lat/lon as zero and just set the location string
		alert.Latitude = 0.0
		alert.Longitude = 0.0
		
		// Extract region and country if possible
		g.extractRegionAndCountry(alert, loc)
	}
	
	return nil
}

// extractRegionAndCountry attempts to extract region and country from location
func (g *Geocoder) extractRegionAndCountry(alert *models.Alert, location string) {
	// This is a simplified implementation
	// In production, you would use a proper geocoding service
	
	locationLower := strings.ToLower(location)
	
	// Simple country detection based on common patterns
	countryPatterns := map[string]string{
		"usa":     "United States",
		"us":      "United States", 
		"uk":      "United Kingdom",
		"ca":      "Canada",
		"mx":      "Mexico",
		"de":      "Germany",
		"fr":      "France",
		"it":      "Italy",
		"es":      "Spain",
		"jp":      "Japan",
		"cn":      "China",
		"in":      "India",
		"br":      "Brazil",
		"au":      "Australia",
	}
	
	// Tokenize location to avoid substring collisions (e.g., "es" in "los angeles")
	tokens := regexp.MustCompile(`[^a-z]+`).Split(locationLower, -1)
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, tk := range tokens {
		if tk == "" { continue }
		tokenSet[tk] = struct{}{}
	}
	for pattern, country := range countryPatterns {
		if _, ok := tokenSet[pattern]; ok {
			alert.Country = country
			break
		}
	}
	
	// Simple region detection
	if alert.Country != "" {
		regionMap := map[string]string{
			"United States": "North America",
			"Canada":       "North America", 
			"Mexico":       "North America",
			"United Kingdom": "Europe",
			"Germany":      "Europe",
			"France":       "Europe",
			"Italy":        "Europe",
			"Spain":        "Europe",
			"Japan":        "Asia",
			"China":        "Asia",
			"India":        "Asia",
			"Brazil":       "South America",
			"Australia":    "Oceania",
		}
		
		if region, exists := regionMap[alert.Country]; exists {
			alert.Region = region
		}
	}
}