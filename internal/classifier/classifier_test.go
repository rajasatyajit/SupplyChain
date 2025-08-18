package classifier

import (
	"testing"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

func TestClassifier_Classify(t *testing.T) {
	classifier := New()

	tests := []struct {
		name              string
		alert             models.Alert
		expectedSeverity  string
		expectedSentiment string
	}{
		{
			name: "High severity alert",
			alert: models.Alert{
				Title:   "Major Strike Shuts Down Port",
				Summary: "Critical disruption affecting all operations",
			},
			expectedSeverity:  "high",
			expectedSentiment: "negative",
		},
		{
			name: "Medium severity alert",
			alert: models.Alert{
				Title:   "Traffic Delay on Highway",
				Summary: "Moderate congestion causing delays",
			},
			expectedSeverity:  "medium",
			expectedSentiment: "neutral",
		},
		{
			name: "Low severity alert",
			alert: models.Alert{
				Title:   "Weather Update",
				Summary: "Clear skies expected",
			},
			expectedSeverity:  "low",
			expectedSentiment: "neutral",
		},
		{
			name: "Positive sentiment alert",
			alert: models.Alert{
				Title:   "Port Operations Restored",
				Summary: "All systems back to normal after successful recovery",
			},
			expectedSeverity:  "low",
			expectedSentiment: "positive",
		},
		{
			name: "Negative sentiment alert",
			alert: models.Alert{
				Title:   "Supply Chain Risk Warning",
				Summary: "Potential shortage threatens operations",
			},
			expectedSeverity:  "medium",
			expectedSentiment: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier.Classify(&tt.alert)

			if tt.alert.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, tt.alert.Severity)
			}

			if tt.alert.Sentiment != tt.expectedSentiment {
				t.Errorf("Expected sentiment %s, got %s", tt.expectedSentiment, tt.alert.Sentiment)
			}

			if tt.alert.Confidence == 0 {
				t.Errorf("Expected confidence to be set")
			}
		})
	}
}

func TestClassifier_ClassifySeverity(t *testing.T) {
	classifier := New()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "High severity keywords",
			text:     "emergency shutdown critical failure",
			expected: "high",
		},
		{
			name:     "Medium severity keywords",
			text:     "traffic delay and congestion",
			expected: "medium",
		},
		{
			name:     "Low severity default",
			text:     "normal operations continue",
			expected: "low",
		},
		{
			name:     "Case insensitive",
			text:     "MAJOR EARTHQUAKE DISASTER",
			expected: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.classifySeverity(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClassifier_ClassifySentiment(t *testing.T) {
	classifier := New()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Negative sentiment",
			text:     "disruption and risk causing concern",
			expected: "negative",
		},
		{
			name:     "Positive sentiment",
			text:     "resolved and restored successfully",
			expected: "positive",
		},
		{
			name:     "Neutral sentiment",
			text:     "normal operations continue",
			expected: "neutral",
		},
		{
			name:     "Case insensitive negative",
			text:     "DANGER AND THREAT DETECTED",
			expected: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.classifySentiment(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}