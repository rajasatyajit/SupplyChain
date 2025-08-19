package utils

import (
	"testing"
)

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		expected bool
	}{
		{
			name:     "Contains one keyword",
			text:     "This is a test message with error",
			keywords: []string{"error", "warning", "failure"},
			expected: true,
		},
		{
			name:     "Contains multiple keywords",
			text:     "System failure detected with error code",
			keywords: []string{"error", "warning", "failure"},
			expected: true,
		},
		{
			name:     "Contains no keywords",
			text:     "This is a normal message",
			keywords: []string{"error", "warning", "failure"},
			expected: false,
		},
		{
			name:     "Case sensitive match",
			text:     "This has ERROR in caps",
			keywords: []string{"error", "warning", "failure"},
			expected: false,
		},
		{
			name:     "Case sensitive match - exact case",
			text:     "This has error in lowercase",
			keywords: []string{"error", "warning", "failure"},
			expected: true,
		},
		{
			name:     "Empty keywords",
			text:     "Any text here",
			keywords: []string{},
			expected: false,
		},
		{
			name:     "Empty text",
			text:     "",
			keywords: []string{"error", "warning"},
			expected: false,
		},
		{
			name:     "Partial word match",
			text:     "This is an errors message",
			keywords: []string{"error"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsAny(tt.text, tt.keywords)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestInferDisruption(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Port disruption",
			text:     "Strike at the port facility",
			expected: "port_status",
		},
		{
			name:     "Port disruption - case insensitive",
			text:     "Issues at PORT OF LOS ANGELES",
			expected: "port_status",
		},
		{
			name:     "Rail disruption",
			text:     "Railway maintenance causing delays",
			expected: "rail",
		},
		{
			name:     "Road disruption - truck",
			text:     "Truck accident on highway",
			expected: "road",
		},
		{
			name:     "Road disruption - road",
			text:     "Road closure due to construction",
			expected: "road",
		},
		{
			name:     "Air disruption - air",
			text:     "Air traffic delays reported",
			expected: "air",
		},
		{
			name:     "Air disruption - airport",
			text:     "Airport security incident",
			expected: "air",
		},
		{
			name:     "General disruption",
			text:     "Supply chain issues affecting delivery",
			expected: "general",
		},
		{
			name:     "Empty text",
			text:     "",
			expected: "general",
		},
		{
			name:     "Multiple keywords - first match wins",
			text:     "Port and rail disruptions reported",
			expected: "port_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferDisruption(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func BenchmarkContainsAny(b *testing.B) {
	text := "This is a long text message that contains various keywords and phrases that we need to search through for performance testing"
	keywords := []string{"error", "warning", "failure", "critical", "emergency", "alert", "issue", "problem"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ContainsAny(text, keywords)
	}
}

func BenchmarkInferDisruption(b *testing.B) {
	text := "Major port strike affecting rail and road transportation with air traffic delays"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InferDisruption(text)
	}
}
