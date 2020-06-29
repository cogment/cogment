package api

type SearchLogsResult struct {
	Results []*LogEntry `json:"results"`
	Count   int         `json:"count"`
	Total   int         `json:"total"`
}

type LogEntry struct {
	ServiceName string  `json:"service_name"`
	Message     string  `json:"message"`
	Timestamp   float64 `json:"timestamp"`
	Source      string  `json:"source"`
}
