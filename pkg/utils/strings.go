package utils

import "strings"

// ContainsAny checks if the text contains any of the given keywords
func ContainsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// InferDisruption infers the disruption type from text
func InferDisruption(text string) string {
	text = strings.ToLower(text)
switch {
	case strings.Contains(text, "airport"):
		return "air"
	case strings.Contains(text, "air"):
		return "air"
	case strings.Contains(text, "port"):
		return "port_status"
	case strings.Contains(text, "rail"):
		return "rail"
	case strings.Contains(text, "truck") || strings.Contains(text, "road"):
		return "road"
	default:
		return "general"
	}
}