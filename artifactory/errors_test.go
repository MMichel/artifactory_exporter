package artifactory

import (
	"testing"
)

func TestUnmarshalError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		endpoint string
	}{
		{
			name:     "Basic error",
			message:  "unmarshal error",
			endpoint: "/api/test",
		},
		{
			name:     "Empty message",
			message:  "",
			endpoint: "/api/test",
		},
		{
			name:     "Empty endpoint",
			message:  "unmarshal error",
			endpoint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &UnmarshalError{
				message:  tt.message,
				endpoint: tt.endpoint,
			}

			// Test Error() method
			if got := err.Error(); got != tt.message {
				t.Errorf("UnmarshalError.Error() = %v, want %v", got, tt.message)
			}

			// Test apiEndpoint() method
			if got := err.apiEndpoint(); got != tt.endpoint {
				t.Errorf("UnmarshalError.apiEndpoint() = %v, want %v", got, tt.endpoint)
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		endpoint string
		status   int
	}{
		{
			name:     "Basic error",
			message:  "API error",
			endpoint: "/api/test",
			status:   404,
		},
		{
			name:     "Empty message",
			message:  "",
			endpoint: "/api/test",
			status:   500,
		},
		{
			name:     "Empty endpoint",
			message:  "API error",
			endpoint: "",
			status:   400,
		},
		{
			name:     "Zero status",
			message:  "API error",
			endpoint: "/api/test",
			status:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{
				message:  tt.message,
				endpoint: tt.endpoint,
				status:   tt.status,
			}

			// Test Error() method
			if got := err.Error(); got != tt.message {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.message)
			}

			// Test apiEndpoint() method
			if got := err.apiEndpoint(); got != tt.endpoint {
				t.Errorf("APIError.apiEndpoint() = %v, want %v", got, tt.endpoint)
			}

			// Test apiStatus() method
			if got := err.apiStatus(); got != tt.status {
				t.Errorf("APIError.apiStatus() = %v, want %v", got, tt.status)
			}
		})
	}
} 