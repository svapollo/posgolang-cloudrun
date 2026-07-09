package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeZipcodeClient struct {
	city string
	err  error
}

func (f fakeZipcodeClient) City(_ context.Context, _ string) (string, error) {
	return f.city, f.err
}

type fakeWeatherClient struct {
	temp float64
	err  error
}

func (f fakeWeatherClient) TemperatureC(_ context.Context, _ string) (float64, error) {
	return f.temp, f.err
}

func TestTemperatureResponseFromC(t *testing.T) {
	got := temperatureResponseFromC(28.5)

	if got.TempC != 28.5 {
		t.Fatalf("TempC = %v, want 28.5", got.TempC)
	}
	if got.TempF != 83.3 {
		t.Fatalf("TempF = %v, want 83.3", got.TempF)
	}
	if got.TempK != 301.65 {
		t.Fatalf("TempK = %v, want 301.65", got.TempK)
	}
}

func TestServeHTTPSuccess(t *testing.T) {
	app := NewApp(
		fakeZipcodeClient{city: "Sao Paulo"},
		fakeWeatherClient{temp: 28.5},
	)

	req := httptest.NewRequest(http.MethodGet, "/?cep=01001000", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got temperatureResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := temperatureResponseFromC(28.5)
	if got != want {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestServeHTTPInvalidZipcode(t *testing.T) {
	app := NewApp(fakeZipcodeClient{}, fakeWeatherClient{})

	req := httptest.NewRequest(http.MethodGet, "/?cep=123", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if got := rr.Body.String(); got != "invalid zipcode" {
		t.Fatalf("body = %q, want %q", got, "invalid zipcode")
	}
}

func TestServeHTTPZipcodeNotFound(t *testing.T) {
	app := NewApp(
		fakeZipcodeClient{err: errZipcodeNotFound},
		fakeWeatherClient{},
	)

	req := httptest.NewRequest(http.MethodGet, "/?cep=99999999", nil)
	rr := httptest.NewRecorder()

	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Body.String(); got != "can not find zipcode" {
		t.Fatalf("body = %q, want %q", got, "can not find zipcode")
	}
}

func TestViaCEPClientCity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/01001000/json/" {
			t.Fatalf("path = %q, want /ws/01001000/json/", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "" {
			t.Fatalf("unexpected query q = %q", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"localidade": "Sao Paulo",
		})
	}))
	defer server.Close()

	client := NewZipcodeClient(server.Client(), server.URL)
	city, err := client.City(context.Background(), "01001000")
	if err != nil {
		t.Fatalf("City() error = %v", err)
	}
	if city != "Sao Paulo" {
		t.Fatalf("city = %q, want %q", city, "Sao Paulo")
	}
}

func TestWeatherClientTemperatureC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/current.json" {
			t.Fatalf("path = %q, want /current.json", r.URL.Path)
		}
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Fatalf("key = %q, want %q", got, "test-key")
		}
		if got := r.URL.Query().Get("q"); got != "Sao Paulo" {
			t.Fatalf("q = %q, want %q", got, "Sao Paulo")
		}
		if got := r.URL.Query().Get("aqi"); got != "no" {
			t.Fatalf("aqi = %q, want no", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{
				"temp_c": 28.5,
			},
		})
	}))
	defer server.Close()

	client := NewWeatherClient(server.Client(), server.URL, "test-key")
	temp, err := client.TemperatureC(context.Background(), "Sao Paulo")
	if err != nil {
		t.Fatalf("TemperatureC() error = %v", err)
	}
	if temp != 28.5 {
		t.Fatalf("temp = %v, want 28.5", temp)
	}
}
