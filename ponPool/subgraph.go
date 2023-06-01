package ponpool

import "net/http"

var (
	BuilderRequest  = "{\"query\":\"{\\n  builders(first:1000){\\n    id\\n    status\\n  }\\n}\",\"variables\":{}}"
	ProposerRequest = "{\"query\":\"{\\n  proposers(first:1000){\\n    id\\n    status\\n    reportCount\\n  }\\n}\",\"variables\":{}}"
	ReporterRequest = "{\"query\":\"{\\n  reporters(first:1000){\\n    id\\n    active\\n    numberOfReports\\n  }\\n}\",\"variables\":{}}"
)

type PonRegistrySubgraph struct {
	Client  http.Client
	URL     string
	API_KEY string
}
