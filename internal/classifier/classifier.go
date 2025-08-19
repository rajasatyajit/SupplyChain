package classifier

import (
	"strings"

	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/pkg/utils"
)

// Classifier provides alert classification functionality
type Classifier struct{}

// New creates a new classifier instance
func New() *Classifier {
	return &Classifier{}
}

// Classify analyzes and classifies an alert
func (c *Classifier) Classify(alert *models.Alert) {
	text := strings.ToLower(alert.Title + " " + alert.Summary)

	// Classify severity
	alert.Severity = c.classifySeverity(text)

	// Classify sentiment
	alert.Sentiment = c.classifySentiment(text)

	// Set initial confidence
	if alert.Confidence == 0 {
		alert.Confidence = 0.8 // Default confidence
	}
}

// classifySeverity determines the severity level of an alert
func (c *Classifier) classifySeverity(text string) string {
	text = strings.ToLower(text)

	highSeverityKeywords := []string{
		"strike", "shutdown", "closure", "blocked", "riot",
		"earthquake", "hurricane", "emergency", "critical",
		"severe", "major", "catastrophic", "disaster",
	}

	mediumSeverityKeywords := []string{
		"delay", "congestion", "backlog", "maintenance",
		"disruption", "issue", "problem", "warning",
		"moderate", "minor",
	}

	if utils.ContainsAny(text, highSeverityKeywords) {
		return "high"
	} else if utils.ContainsAny(text, mediumSeverityKeywords) {
		return "medium"
	}

	return "low"
}

// classifySentiment determines the sentiment of an alert
func (c *Classifier) classifySentiment(text string) string {
	text = strings.ToLower(text)

	negativeKeywords := []string{
		"disrupt", "risk", "shortage", "warning", "danger",
		"threat", "crisis", "failure", "damage", "loss",
		"concern", "worry", "fear", "panic",
	}

	positiveKeywords := []string{
		"resolved", "fixed", "restored", "improved",
		"success", "recovery", "solution", "progress",
	}

	if utils.ContainsAny(text, negativeKeywords) {
		return "negative"
	} else if utils.ContainsAny(text, positiveKeywords) {
		return "positive"
	}

	return "neutral"
}
