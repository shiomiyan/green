package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const switchBotAPIV11 = "https://api.switch-bot.com"
const (
	switchBotTokenEnv         = "SWITCHBOT_TOKEN"
	switchBotClientSecretEnv  = "SWITCHBOT_CLIENT_SECRET"
	switchBotMeterDeviceIDEnv = "SWITCHBOT_METERPLUS_DEVICE_ID"
	localVictoriaMetricsURL   = "http://127.0.0.1:8428"
	victoriaMetricsWritePath  = "/api/v1/import/prometheus"
)

type environment struct {
	token              string
	secret             string
	deviceID           string
	baseURL            string
	victoriaMetricsURL string
	client             *http.Client
}

type switchBotStatusResponse struct {
	StatusCode int                  `json:"statusCode"`
	Message    string               `json:"message"`
	Body       switchBotClimateBody `json:"body"`
}

type switchBotClimateBody struct {
	DeviceID    string  `json:"deviceId"`
	DeviceType  string  `json:"deviceType"`
	Temperature float64 `json:"temperature"`
	Humidity    int     `json:"humidity"`
}

func main() {
	env, err := loadEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load environment: %v\n", err)
		os.Exit(1)
	}

	if err := run(context.Background(), env); err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch climate: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, env environment) error {
	client := env.client
	if client == nil {
		client = http.DefaultClient
	}

	baseURL := env.baseURL
	if baseURL == "" {
		baseURL = switchBotAPIV11
	}

	status, err := fetchClimate(ctx, client, baseURL, env)
	if err != nil {
		return err
	}

	if err := writeClimateMetrics(ctx, client, env.victoriaMetricsURL, status); err != nil {
		return err
	}

	return nil
}

func fetchClimate(ctx context.Context, client *http.Client, baseURL string, env environment) (switchBotClimateBody, error) {
	nonce, err := randomNonce()
	if err != nil {
		return switchBotClimateBody{}, fmt.Errorf("generate nonce: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	url := fmt.Sprintf("%s/v1.1/devices/%s/status", strings.TrimRight(baseURL, "/"), env.deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return switchBotClimateBody{}, fmt.Errorf("build request: %w", err)
	}
	req.Header = newAuthHeaders(env.token, env.secret, timestamp, nonce)

	resp, err := client.Do(req)
	if err != nil {
		return switchBotClimateBody{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return switchBotClimateBody{}, fmt.Errorf("unexpected http status: %s", resp.Status)
	}

	var payload switchBotStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return switchBotClimateBody{}, fmt.Errorf("decode response: %w", err)
	}
	if payload.StatusCode != 100 {
		return switchBotClimateBody{}, fmt.Errorf("switchbot api error: statusCode=%d message=%q", payload.StatusCode, payload.Message)
	}

	return payload.Body, nil
}

func loadEnvironment() (environment, error) {
	env := environment{
		token:              os.Getenv(switchBotTokenEnv),
		secret:             os.Getenv(switchBotClientSecretEnv),
		deviceID:           os.Getenv(switchBotMeterDeviceIDEnv),
		victoriaMetricsURL: localVictoriaMetricsURL,
	}
	if env.token == "" {
		return environment{}, errors.New(switchBotTokenEnv + " is required")
	}
	if env.secret == "" {
		return environment{}, errors.New(switchBotClientSecretEnv + " is required")
	}
	if env.deviceID == "" {
		return environment{}, errors.New(switchBotMeterDeviceIDEnv + " is required")
	}
	return env, nil
}

func writeClimateMetrics(ctx context.Context, client *http.Client, baseURL string, climate switchBotClimateBody) error {
	body := climateMetricsPayload(climate)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+victoriaMetricsWritePath,
		bytes.NewBufferString(body),
	)
	if err != nil {
		return fmt.Errorf("build victoria metrics request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; version=0.0.4")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send victoria metrics request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		return nil
	}

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("victoria metrics write failed: status=%s", resp.Status)
	}
	if len(responseBody) == 0 {
		return fmt.Errorf("victoria metrics write failed: status=%s", resp.Status)
	}

	return fmt.Errorf("victoria metrics write failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(responseBody)))
}

func climateMetricsPayload(climate switchBotClimateBody) string {
	labels := metricLabels(climate)
	humidityRatio := float64(climate.Humidity) / 100

	return "" +
		"switchbot_meterplus_1_temperature_celsius" + labels + " " + strconv.FormatFloat(climate.Temperature, 'f', -1, 64) + "\n" +
		"switchbot_meterplus_1_humidity_ratio" + labels + " " + strconv.FormatFloat(humidityRatio, 'f', -1, 64) + "\n"
}

func metricLabels(climate switchBotClimateBody) string {
	return "{source=\"meterplus\"" +
		",device_id=\"" + climate.DeviceID +
		"\",device_type=\"" + climate.DeviceType + "\"}"
}

func newAuthHeaders(token string, secret string, timestamp string, nonce string) http.Header {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token + timestamp + nonce))
	signature := strings.ToUpper(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	headers := make(http.Header)
	headers.Set("Authorization", token)
	headers.Set("Content-Type", "application/json")
	headers.Set("charset", "utf8")
	headers.Set("t", timestamp)
	headers.Set("sign", signature)
	headers.Set("nonce", nonce)

	return headers
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
