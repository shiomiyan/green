package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRunPostsClimateToVictoriaMetrics(t *testing.T) {
	t.Parallel()

	var gotPostPath string
	var gotPostBody string

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.1/devices/meter-1/status":
			if got := r.Header.Get("Authorization"); got != "test-token" {
				t.Fatalf("Authorization = %q", got)
			}
			if got := r.Header.Get("sign"); got == "" {
				t.Fatal("sign header is empty")
			}
			if got := r.Header.Get("nonce"); got == "" {
				t.Fatal("nonce header is empty")
			}
			if got := r.Header.Get("t"); got == "" {
				t.Fatal("t header is empty")
			}

			return jsonResponse(
				`{"statusCode":100,"message":"success","body":{"deviceId":"meter-1","deviceType":"Meter Plus","temperature":24.6,"humidity":51}}`,
			), nil
		case "/api/v1/import/prometheus":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("io.ReadAll() error = %v", err)
			}
			gotPostPath = r.URL.Path
			gotPostBody = string(body)

			return &http.Response{
				StatusCode: http.StatusNoContent,
				Status:     "204 No Content",
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})}

	err := run(context.Background(), environment{
		token:              "test-token",
		secret:             "test-secret",
		deviceID:           "meter-1",
		baseURL:            "https://example.invalid",
		victoriaMetricsURL: "https://vm.example.invalid/",
		client:             client,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got, want := gotPostPath, "/api/v1/import/prometheus"; got != want {
		t.Fatalf("POST path = %q, want %q", got, want)
	}

	wantBody := "" +
		"switchbot_meterplus_1_temperature_celsius{source=\"meterplus\",device_id=\"meter-1\",device_type=\"Meter Plus\"} 24.6\n" +
		"switchbot_meterplus_1_humidity_ratio{source=\"meterplus\",device_id=\"meter-1\",device_type=\"Meter Plus\"} 0.51\n"
	if gotPostBody != wantBody {
		t.Fatalf("POST body = %q, want %q", gotPostBody, wantBody)
	}
}

func TestRunFailsWhenVictoriaMetricsReturnsError(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.1/devices/meter-1/status":
			return jsonResponse(
				`{"statusCode":100,"message":"success","body":{"deviceId":"meter-1","deviceType":"Meter","temperature":24.6,"humidity":51}}`,
			), nil
		case "/api/v1/import/prometheus":
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Body:       io.NopCloser(strings.NewReader("bad metrics")),
			}, nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})}

	err := run(context.Background(), environment{
		token:              "test-token",
		secret:             "test-secret",
		deviceID:           "meter-1",
		baseURL:            "https://example.invalid",
		victoriaMetricsURL: "https://vm.example.invalid",
		client:             client,
	})
	if err == nil {
		t.Fatal("run() error = nil")
	}
	if !strings.Contains(err.Error(), "bad metrics") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadEnvironmentReadsProcessEnvironment(t *testing.T) {
	t.Setenv(switchBotTokenEnv, "from-env-token")
	t.Setenv(switchBotClientSecretEnv, "from-env-secret")
	t.Setenv(switchBotMeterDeviceIDEnv, "from-env-device")

	env, err := loadEnvironment()
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}

	if env.token != "from-env-token" {
		t.Fatalf("token = %q", env.token)
	}
	if env.secret != "from-env-secret" {
		t.Fatalf("secret = %q", env.secret)
	}
	if env.deviceID != "from-env-device" {
		t.Fatalf("deviceID = %q", env.deviceID)
	}
	if env.victoriaMetricsURL != "http://127.0.0.1:8428" {
		t.Fatalf("victoriaMetricsURL = %q", env.victoriaMetricsURL)
	}
}

func TestLoadEnvironmentFailsWhenRequiredValueMissing(t *testing.T) {
	t.Setenv(switchBotTokenEnv, "only-token")

	_, err := loadEnvironment()
	if err == nil {
		t.Fatal("loadEnvironment() error = nil")
	}
	if !strings.Contains(err.Error(), switchBotClientSecretEnv) {
		t.Fatalf("error = %q", err)
	}
}

func TestSignatureUsesDocumentedAlgorithm(t *testing.T) {
	t.Parallel()

	headers := newAuthHeaders("test-token", "test-secret", "1700000000123", "request-id")

	if got, want := headers.Get("Authorization"), "test-token"; got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
	if got, want := headers.Get("t"), "1700000000123"; got != want {
		t.Fatalf("t = %q, want %q", got, want)
	}
	if got, want := headers.Get("nonce"), "request-id"; got != want {
		t.Fatalf("nonce = %q, want %q", got, want)
	}
	if got, want := headers.Get("sign"), "CHHA6OIR2PAT9FJOGCD7USIUIFUEXJIKF7MU2WEZL98="; got != want {
		t.Fatalf("sign = %q, want %q", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
