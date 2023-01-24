package internal

var Environment = "dev"
var Hash string

const (
	ProdEndpoint = "https://app.metrist.io/api/v0"
	Dev1Endpoint = "https://app-dev1.metrist.io/api/v0"
)

func Endpoint() string {
	switch Environment {
	case "prod":
		return ProdEndpoint
	default:
		return Dev1Endpoint
	}
}
