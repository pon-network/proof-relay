package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (b *beaconClient) fetchBeacon(u *url.URL, dst any) error {
	/*
		Utility function to fetch data from the beacon node
	*/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("invalid request for %s: %w", u, err)
	}
	req.Header.Set("accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("client refused for %s: %w", u, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body for %s: %w", u, err)
	}

	if resp.StatusCode >= 300 {
		ec := &struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal(bodyBytes, ec); err != nil {
			return fmt.Errorf("could not unmarshal error response from beacon node for %s from %s: %w", u, string(bodyBytes), err)
		}
		return fmt.Errorf("error response from beacon node for %s: %s", u, ec.Message)
	}

	err = json.Unmarshal(bodyBytes, dst)
	if err != nil {
		return fmt.Errorf("could not unmarshal response for %s from %s: %w", u, string(bodyBytes), err)
	}

	return nil
}

func (b *beaconClient) postBeacon(u *url.URL, src any) error {
	/*
		Utility function to post data to the beacon node
	*/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// check if src is bytes or not
	var src_bytes []byte
	if _, ok := src.([]byte); !ok {
		src_json, err := json.Marshal(src)
		if err != nil {
			return fmt.Errorf("could not marshal payload: %w", err)
		}
		src_bytes = src_json
	} else {
		src_bytes = src.([]byte)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(src_bytes))
	if err != nil {
		return fmt.Errorf("invalid request for %s: %w", u, err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post request: %w", err)
	}

	if resp.StatusCode >= 300 {
		ec := &struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response body: %w", err)
		}

		if err = json.Unmarshal(bodyBytes, ec); err != nil {
			return fmt.Errorf("failed to unmarshal error response: %w", err)
		}
		return fmt.Errorf("error response from beacon node for %s: %s", u, ec.Message)
	}

	return nil
}
