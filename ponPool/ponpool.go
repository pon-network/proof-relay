package ponpool

import (
	"encoding/json"
	"errors"
	"math/big"
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

func (s *PonRegistrySubgraph) GetBuilders() ([]ponPoolTypes.BuilderInterface, error) {
	payload := strings.NewReader(BuilderRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	builderResponse := new(ponPoolTypes.BuilderPool)
	if err := json.NewDecoder(res.Body).Decode(&builderResponse); err != nil {
		return nil, err
	}

	builderStakeRequired, err := s.getBuilderRequiredStake()
	if err != nil {
		return nil, err
	}

	builders := *new([]ponPoolTypes.BuilderInterface)

	for _, builder := range builderResponse.Data.Builders {
		builderBalance := new(big.Int)
		builderBalance, ok := builderBalance.SetString(builder.BalanceStaked, 10)

		status := true
		if statusCompare := builderBalance.Cmp(builderStakeRequired); statusCompare < 0 {
			status = false
		}

		if !ok {
			return nil, errors.New("Failed to get the BigInt Builder Stake")
		}
		builders = append(builders, ponPoolTypes.BuilderInterface{
			Builder: builder,
			Status:  status,
		})

	}

	return builders, nil
}

func (s *PonRegistrySubgraph) getBuilderRequiredStake() (*big.Int, error) {
	payload := strings.NewReader(BuilderStakeRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	builderResponse := new(ponPoolTypes.BuilderStake)
	if err := json.NewDecoder(res.Body).Decode(&builderResponse); err != nil {
		return nil, err
	}
	builderStake := new(big.Int)
	builderStake, ok := builderStake.SetString(builderResponse.Data.GlobalValue.BalanceRequired, 10)
	if !ok {
		return nil, errors.New("Failed to get the BigInt Builder Stake")
	}
	return builderStake, nil
}

func (s *PonRegistrySubgraph) GetValidators() ([]ponPoolTypes.Validator, error) {
	payload := strings.NewReader(ProposerRequest)
	req, err := http.NewRequest("POST", s.URL, payload)
	if err != nil {

		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
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
	req.Header.Add("Content-Type", "application/json")
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
