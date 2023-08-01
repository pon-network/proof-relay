package ponpool

import (
	"encoding/json"
	"net/http"
	"strings"

	ponPoolTypes "github.com/bsn-eng/pon-golang-types/ponPool"
)

func NewPonPool(url string, apiKey string) *PonRegistrySubgraph {
	return &PonRegistrySubgraph{
		Client:  http.Client{},
		URL:     url,
		API_KEY: apiKey,
	}
}

func (s *PonRegistrySubgraph) GetBuilders() ([]ponPoolTypes.Builder, error) {
	payload := strings.NewReader(BuilderRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {
		return nil, err
	}
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	builderResponse := new(ponPoolTypes.BuilderPool)
	if err := json.NewDecoder(res.Body).Decode(&builderResponse); err != nil {
		return nil, err
	}
	return builderResponse.Data.Builders, nil
}

func (s *PonRegistrySubgraph) GetValidators() ([]ponPoolTypes.Validator, error) {
	payload := strings.NewReader(ProposerRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {

		return nil, err
	}
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	validatorResponse := new(ponPoolTypes.ValidatorPool)
	if err := json.NewDecoder(res.Body).Decode(&validatorResponse); err != nil {
		return nil, err
	}

	return validatorResponse.Data.Validators, nil
}

func (s *PonRegistrySubgraph) GetReporters() ([]ponPoolTypes.Reporter, error) {
	payload := strings.NewReader(ReporterRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {

		return nil, err
	}
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	reporterResponse := new(ponPoolTypes.ReporterPool)
	if err := json.NewDecoder(res.Body).Decode(&reporterResponse); err != nil {
		return nil, err
	}
	return reporterResponse.Data.Reporters, nil
}
