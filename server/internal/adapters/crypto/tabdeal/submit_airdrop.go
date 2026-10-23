package tabdeal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type SubmitAirdropRequest struct {
	URL             string `json:"url"`
	SocialMediaCode string `json:"social_media_code"`
	CurrencySymbol  string `json:"currency_symbol"`
}

type SubmitAirdropResponse struct {
	Error    string `json:"error"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
}

func (a *Adapter) SubmitAirdrop(ctx context.Context, req SubmitAirdropRequest) (*SubmitAirdropResponse, error) {
	baseURL := strings.TrimRight(a.cfg.APIBaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("wallex api base url is empty")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/festival/instagram-prize-verification/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authToken := a.getAuthToken(); authToken != "" {
		httpReq.Header.Set("Authorization", authToken)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SubmitAirdropResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && resp.StatusCode < http.StatusBadRequest {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{StatusCode: resp.StatusCode, Body: result.Error}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("wallex airdrop submit failed: status=%d message=%s", resp.StatusCode, result.Error)
	}
	if result.Currency == "" {
		result.Currency = req.CurrencySymbol
	}
	return &result, nil
}
