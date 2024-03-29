package internal

var Environment = "dev"
var BuildHash string

const (
	ProdEndpoint  = "https://app.metrist.io"
	Dev1Endpoint  = "https://app-dev1.metrist.io"
	LocalEndpoint = "https://host.docker.internal:4443"
)

func Endpoint() string {
	switch Environment {
	case "prod":
		return ProdEndpoint
	case "local":
		return LocalEndpoint
	default:
		return Dev1Endpoint
	}
}
