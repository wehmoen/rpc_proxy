package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"os"
	"time"
)

// TrackingEvent represents a tracking event.
type TrackingEvent struct {
	UUID             string            `json:"uuid,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	UserID           string            `json:"user_id,omitempty"`
	Event            string            `json:"event"`
	Ref              string            `json:"ref"`
	Timestamp        string            `json:"timestamp"`
	Action           string            `json:"action"`
	ActionProperties map[string]string `json:"action_properties,omitempty"`
}

type eventWrapper struct {
	Type string        `json:"type"`
	Data TrackingEvent `json:"data"`
}

type trackingRequest struct {
	Events []eventWrapper `json:"events"`
	ApiKey string         `json:"api_key"`
}

// SkyMavisTracking represents the SkyMavis tracking object.
type SkyMavisTracking struct {
	apiKey string
	client *resty.Client
}

// NewSkyMavisTracking creates a new SkyMavisTracking object.
func NewSkyMavisTracking(apiKey string) *SkyMavisTracking {
	client := resty.New()
	return &SkyMavisTracking{apiKey: apiKey, client: client}
}

// Send sends a tracking event.
func (s *SkyMavisTracking) Send(event TrackingEvent) (*resty.Response, error) {
	event.UUID = s.generateUUID(event.Ref)
	event.SessionID = s.generateUUID(event.UUID)
	event.UserID = s.generateUUID(event.SessionID)

	basicAuth := base64.StdEncoding.EncodeToString([]byte(s.apiKey + ":"))

	resp, err := s.client.R().
		SetHeader("Authorization", "Basic "+basicAuth).
		SetHeader("Content-Type", "application/json").
		SetBody(trackingRequest{
			ApiKey: s.apiKey,
			Events: []eventWrapper{
				{
					Type: "track",
					Data: event,
				},
			},
		}).
		Post("https://x.skymavis.com/track")

	if os.Getenv("DEBUG") == "true" {
		println("Status code: " + resp.Status())
		println("Result: " + string(resp.Body()))
	}

	return resp, err
}

// GenerateUUID generates a UUID.
func (s *SkyMavisTracking) generateUUID(input string) string {
	hash := sha256.Sum256([]byte(input))
	randomPart := hash[:16]

	newUUID, err := uuid.NewRandomFromReader(bytes.NewReader(randomPart))
	if err != nil {
		return ""
	}
	return newUUID.String()
}

// TrackAPIRequest tracks an API request.
func (s *SkyMavisTracking) TrackAPIRequest(ipAddress string, route string, properties map[string]string) (*resty.Response, error) {
	event := TrackingEvent{
		Event:     "api_request",
		Ref:       ipAddress,
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    route,
	}
	if properties != nil {
		event.ActionProperties = properties
	}
	return s.Send(event)
}
