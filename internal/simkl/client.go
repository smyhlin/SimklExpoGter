package simkl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	AllowedTypes          = []string{"shows", "movies", "anime"}
	AllowedStatuses       = []string{"watching", "plantowatch", "hold", "completed", "dropped"}
	AllowedExtendedValues = []string{"full", "full_anime_seasons", "simkl_ids_only", "ids_only"}
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type DeviceCodeResponse struct {
	Result          string `json:"result"`
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DeviceCodeStatusResponse struct {
	Result      string `json:"result"`
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}

type FetchRequest struct {
	Type                 string
	Status               string
	DateFrom             string
	Extended             string
	EpisodeWatchedAt     bool
	IncludeMemos         bool
	IncludeNextWatchInfo bool
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func NewClient() *Client {
	return NewClientWithBaseURL("https://api.simkl.com", &http.Client{
		Timeout: 45 * time.Second,
	})
}

func NewClientWithBaseURL(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 45 * time.Second,
		}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) RequestDeviceCode(ctx context.Context, clientID, redirect string) (DeviceCodeResponse, error) {
	endpoint, err := url.Parse(c.baseURL + "/oauth/pin")
	if err != nil {
		return DeviceCodeResponse{}, err
	}

	query := endpoint.Query()
	query.Set("client_id", clientID)
	if strings.TrimSpace(redirect) != "" {
		query.Set("redirect", redirect)
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return DeviceCodeResponse{}, err
	}

	var response DeviceCodeResponse
	if err := c.doJSON(request, &response, ""); err != nil {
		return DeviceCodeResponse{}, err
	}

	return response, nil
}

func (c *Client) PollDeviceCode(ctx context.Context, clientID, userCode string) (DeviceCodeStatusResponse, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/oauth/pin/%s", c.baseURL, url.PathEscape(userCode)))
	if err != nil {
		return DeviceCodeStatusResponse{}, err
	}

	query := endpoint.Query()
	query.Set("client_id", clientID)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return DeviceCodeStatusResponse{}, err
	}

	var response DeviceCodeStatusResponse
	if err := c.doJSON(request, &response, ""); err != nil {
		return DeviceCodeStatusResponse{}, err
	}

	return response, nil
}

func (c *Client) ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (TokenResponse, error) {
	endpoint := c.baseURL + "/oauth/token"

	payload := map[string]string{
		"code":          code,
		"client_id":     clientID,
		"client_secret": clientSecret,
		"redirect_uri":  redirectURI,
		"grant_type":    "authorization_code",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return TokenResponse{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(data)))
	if err != nil {
		return TokenResponse{}, err
	}

	var response TokenResponse
	if err := c.doJSON(request, &response, ""); err != nil {
		return TokenResponse{}, err
	}

	if response.Error != "" {
		return response, fmt.Errorf("simkl oauth error: %s", response.Error)
	}

	return response, nil
}

func (c *Client) FetchActivities(ctx context.Context, clientID, accessToken string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sync/activities", nil)
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := c.doJSON(request, &response, accessToken, clientID); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) FetchAllItems(ctx context.Context, clientID, accessToken string, fetchRequest FetchRequest) (map[string]any, error) {
	endpoint := strings.TrimRight(c.baseURL, "/") + "/sync/all-items"
	if strings.TrimSpace(fetchRequest.Type) != "" {
		endpoint += "/" + url.PathEscape(fetchRequest.Type)
	}
	if strings.TrimSpace(fetchRequest.Status) != "" {
		endpoint += "/" + url.PathEscape(fetchRequest.Status)
	}
	endpoint += "/"

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	query := parsedURL.Query()
	if strings.TrimSpace(fetchRequest.DateFrom) != "" {
		query.Set("date_from", fetchRequest.DateFrom)
	}
	if strings.TrimSpace(fetchRequest.Extended) != "" {
		query.Set("extended", fetchRequest.Extended)
	}
	if fetchRequest.EpisodeWatchedAt {
		query.Set("episode_watched_at", "yes")
	}
	if fetchRequest.IncludeMemos {
		query.Set("memos", "yes")
	}
	if fetchRequest.IncludeNextWatchInfo {
		query.Set("next_watch_info", "yes")
	}
	parsedURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := c.doJSON(request, &raw, accessToken, clientID); err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}

	response, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected Simkl payload type %T", raw)
	}

	return response, nil
}

func (c *Client) doJSON(request *http.Request, target any, accessToken string, clientID ...string) error {
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	if strings.TrimSpace(accessToken) != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if len(clientID) > 0 && strings.TrimSpace(clientID[0]) != "" {
		request.Header.Set("simkl-api-key", clientID[0])
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("simkl request failed: %s", message)
	}

	if target == nil {
		return nil
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil && err != io.EOF {
		return err
	}

	return nil
}
