package plugin

type queryModel struct {
	QueryType string `json:"queryType"`
}

type monitorTelemetryQuery struct {
	Monitors      []string `json:"monitors"`
	IncludeShared bool     `json:"includeshared"`
}
