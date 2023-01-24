package internal

var Environment = "dev"

const (
	ProdEndpoint = "https://app.metrist.io/api/v0"
	Dev1Endpoint = "https://app-dev1.metrist.io/api/v0"
)

func Endpoint() string {
	switch Environment {
	case "dev":
		return Dev1Endpoint
	default:
		return ProdEndpoint
	}
}
