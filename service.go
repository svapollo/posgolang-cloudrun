package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var cepPattern = regexp.MustCompile(`^\d{8}$`)

var errZipcodeNotFound = errors.New("zipcode not found")

type ZipcodeClient interface {
	City(context.Context, string) (string, error)
}

type WeatherClient interface {
	TemperatureC(context.Context, string) (float64, error)
}

type App struct {
	ZipcodeClient ZipcodeClient
	WeatherClient  WeatherClient
}

func NewApp(zipcodeClient ZipcodeClient, weatherClient WeatherClient) *App {
	return &App{
		ZipcodeClient: zipcodeClient,
		WeatherClient: weatherClient,
	}
}

type temperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cep := extractCEP(r)
	if !isValidCEP(cep) {
		writeText(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	city, err := a.ZipcodeClient.City(r.Context(), cep)
	if err != nil {
		if errors.Is(err, errZipcodeNotFound) {
			writeText(w, http.StatusNotFound, "can not find zipcode")
			return
		}

		writeText(w, http.StatusBadGateway, "failed to resolve zipcode")
		return
	}

	tempC, err := a.WeatherClient.TemperatureC(r.Context(), city)
	if err != nil {
		writeText(w, http.StatusBadGateway, "failed to fetch weather")
		return
	}

	response := temperatureResponseFromC(tempC)
	writeJSON(w, http.StatusOK, response)
}

func extractCEP(r *http.Request) string {
	if cep := strings.TrimSpace(r.URL.Query().Get("cep")); cep != "" {
		return cep
	}

	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		return ""
	}

	parts := strings.Split(path, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func isValidCEP(cep string) bool {
	return cepPattern.MatchString(strings.TrimSpace(cep))
}

func temperatureResponseFromC(tempC float64) temperatureResponse {
	tempC = round2(tempC)
	return temperatureResponse{
		TempC: tempC,
		TempF: round2(tempC*1.8 + 32),
		TempK: round2(tempC + 273.15),
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func writeText(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(payload)
}

type viaCEPClient struct {
	baseURL string
	client  *http.Client
}

func NewZipcodeClient(client *http.Client, baseURL string) ZipcodeClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &viaCEPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *viaCEPClient) City(ctx context.Context, cep string) (string, error) {
	endpoint := c.baseURL + "/ws/" + cep + "/json/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		Localidade string `json:"localidade"`
		Erro       string `json:"erro"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.Erro == "true" || strings.TrimSpace(payload.Localidade) == "" {
		return "", errZipcodeNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", errZipcodeNotFound
	}

	return payload.Localidade, nil
}

type weatherAPIClient struct {
	baseURL string
	client  *http.Client
	apiKey  string
}

func NewWeatherClient(client *http.Client, baseURL, apiKey string) WeatherClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &weatherAPIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		apiKey:  apiKey,
	}
}

func (c *weatherAPIClient) TemperatureC(ctx context.Context, location string) (float64, error) {
	values := url.Values{}
	values.Set("key", c.apiKey)
	values.Set("q", location)
	values.Set("aqi", "no")

	endpoint := c.baseURL + "/current.json?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var payload struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		if payload.Error.Message != "" {
			return 0, errors.New(payload.Error.Message)
		}
		return 0, fmt.Errorf("weatherapi returned status %d", resp.StatusCode)
	}

	var payload struct {
		Current struct {
			TempC float64 `json:"temp_c"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	return payload.Current.TempC, nil
}
