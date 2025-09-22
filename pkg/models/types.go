package models

type NavigationEvent struct {
	VisitorID string `json:"visitor_id"`
	URL       string `json:"url"`
}

type VisitorStats struct {
	URL              string `json:"url"`
	DistinctVisitors int    `json:"distinct_visitors"`
}
