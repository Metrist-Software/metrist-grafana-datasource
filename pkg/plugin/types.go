package plugin

type queryModel struct {
	QueryType string `json:"queryType"`
}

// Right now our query editor share most of the fields
// Once we start having completely different fields for each query, let's start making
// multiple query struct for each query
type monitorTelemetryQuery struct {
	Monitors      []string `json:"monitors"`
	IncludeShared bool     `json:"includeshared"`
}
