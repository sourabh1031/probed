package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gojektech/heimdall/httpclient"
)

type kongClient struct {
	httpClient   *httpclient.Client
	kongAdminURL string
}

func newKongClient(kongHost, kongAdminPort string, timeout time.Duration) *kongClient {
	if timeout == 0 {
		timeout = 1000 * time.Millisecond
	}
	return &kongClient{
		kongAdminURL: fmt.Sprintf("%s:%s", kongHost, kongAdminPort),
		httpClient:   httpclient.NewClient(httpclient.WithHTTPTimeout(timeout)),
	}
}

type upstreamResponse struct {
	Data []upstream `json:"data"`
}

type upstream struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (kc *kongClient) upstreams() ([]upstream, error) {
	upstreams := []upstream{}

	respBytes, err := kc.doRequest(http.MethodGet, "upstreams", nil)
	if err != nil {
		return upstreams, err
	}

	upstreamResponse := &upstreamResponse{}

	err = json.Unmarshal(respBytes, upstreamResponse)
	if err != nil {
		return upstreams, err
	}

	return upstreamResponse.Data, nil
}

type target struct {
	ID         string `json:"id,omitempty"`
	URL        string `json:"target"`
	Weight     int    `json:"weight"`
	UpstreamID string `json:"upstream_id,omitempty"`
}

type targetResponse struct {
	Data []target `json:"data"`
}

func (kc *kongClient) targetsFor(upstreamID string) ([]target, error) {
	targets := []target{}

	respBytes, err := kc.doRequest(http.MethodGet, fmt.Sprintf("upstreams/%s/targets", upstreamID), nil)
	if err != nil {
		return targets, err
	}

	targetResponse := &targetResponse{}

	err = json.Unmarshal(respBytes, targetResponse)
	if err != nil {
		return targets, err
	}

	return targetResponse.Data, nil
}

func (kc *kongClient) setTargetWeightFor(upstreamID, targetURL string, weight int) error {
	target := target{URL: targetURL, Weight: weight}
	requestBody, err := json.Marshal(target)
	if err != nil {
		return err
	}

	_, err = kc.doRequest(http.MethodPost, fmt.Sprintf("upstreams/%s/targets", upstreamID), requestBody)
	if err != nil {
		return err
	}

	return nil
}

func (kc *kongClient) doRequest(method, path string, body []byte) ([]byte, error) {
	var respBytes []byte

	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", kc.kongAdminURL, path), bytes.NewBuffer(body))
	if err != nil {
		return respBytes, fmt.Errorf("failed to create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	response, err := kc.httpClient.Do(req)
	if err != nil {
		return respBytes, err
	}

	defer response.Body.Close()

	respBytes, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return respBytes, err
	}

	if response.StatusCode >= http.StatusBadRequest {
		return respBytes, fmt.Errorf("failed with status: %d", response.StatusCode)
	}

	return respBytes, nil
}
