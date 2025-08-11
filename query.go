package main

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AlertQuery struct {
	Region     string
	Country    string
	Disruption string
	Severity   string
	Since      time.Time
	Limit      int
}

func (q AlertQuery) Matches(a Alert) bool {
	if q.Region != "" && !strings.EqualFold(a.Region, q.Region) { return false }
	if q.Country != "" && !strings.EqualFold(a.Country, q.Country) { return false }
	if q.Disruption != "" && !strings.EqualFold(a.Disruption, q.Disruption) { return false }
	if q.Severity != "" && !strings.EqualFold(a.Severity, q.Severity) { return false }
	if !q.Since.IsZero() && a.DetectedAt.Before(q.Since) { return false }
	return true
}

func ParseAlertQuery(values url.Values) AlertQuery {
	q := AlertQuery{}
	q.Region = values.Get("region")
	q.Country = values.Get("country")
	q.Disruption = values.Get("type")
	q.Severity = values.Get("severity")
	if s := values.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil { q.Since = t }
	}
	if l := values.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 { q.Limit = n }
	}
	return q
}

