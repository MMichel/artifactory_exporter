package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// testEnv represents a test environment configuration
type testEnv struct {
	username    string
	password    string
	accessToken string
	scrapeURI   string
}

// setupTestEnv configures the environment for a test and returns a cleanup function
func setupTestEnv(env testEnv) func() {
	// Save original args
	oldArgs := os.Args
	
	// Set environment variables
	if env.username != "" {
		os.Setenv("ARTI_USERNAME", env.username)
	}
	if env.password != "" {
		os.Setenv("ARTI_PASSWORD", env.password)
	}
	if env.accessToken != "" {
		os.Setenv("ARTI_ACCESS_TOKEN", env.accessToken)
	}
	
	scrapeURI := env.scrapeURI
	if scrapeURI == "" {
		scrapeURI = "http://localhost:8081/artifactory"  // Default valid URI
	}
	os.Setenv("ARTI_SCRAPE_URI", scrapeURI)
	
	// Set minimal args for kingpin
	os.Args = []string{"artifactory_exporter"}
	
	// Return cleanup function
	return func() {
		// Restore original args
		os.Args = oldArgs
		
		// Clean up environment variables
		os.Unsetenv("ARTI_USERNAME")
		os.Unsetenv("ARTI_PASSWORD")
		os.Unsetenv("ARTI_ACCESS_TOKEN")
		os.Unsetenv("ARTI_SCRAPE_URI")
	}
}

// assertError validates expected error conditions with context
func assertError(t *testing.T, err error, expectError bool, context string) {
	t.Helper() // Mark this as a helper function for better test output
	
	if expectError && err == nil {
		t.Errorf("Expected error for %s, but got nil", context)
	}
	if !expectError && err != nil {
		t.Errorf("Expected no error for %s, but got: %v", context, err)
	}
}

// assertEqual compares two values and reports an error if they don't match
func assertEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper() // Mark this as a helper function for better test output
	
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s - Expected: %v, Got: %v", message, expected, actual)
	}
}

// mockOptionalMetrics temporarily replaces optionalMetrics flag value for testing
func mockOptionalMetrics(metrics []string) func() {
	original := *optionalMetrics
	*optionalMetrics = metrics
	return func() {
		*optionalMetrics = original
	}
}

// TestCredentialsValidation verifies the credential validation logic
func TestCredentialsValidation(t *testing.T) {
	testCases := []struct {
		name        string
		username    string
		password    string
		accessToken string
		expectError bool
	}{
		{
			name:        "No credentials",
			expectError: true,
		},
		{
			name:        "Only username provided",
			username:    "user",
			expectError: true,
		},
		{
			name:        "Only password provided",
			password:    "pass",
			expectError: true,
		},
		{
			name:        "Username and password provided",
			username:    "user",
			password:    "pass",
			expectError: false,
		},
		{
			name:        "Access token provided",
			accessToken: "token",
			expectError: false,
		},
		{
			name:        "Both auth methods provided",
			username:    "user",
			password:    "pass",
			accessToken: "token",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupTestEnv(testEnv{
				username:    tc.username,
				password:    tc.password,
				accessToken: tc.accessToken,
			})
			defer cleanup()
			
			_, err := NewConfig()
			
			contextMsg := fmt.Sprintf("credentials: %s/%s/%s", 
				tc.username, tc.password, tc.accessToken)
			assertError(t, err, tc.expectError, contextMsg)
		})
	}
}

// TestAuthMethodDetermination verifies the auth method is correctly determined
func TestAuthMethodDetermination(t *testing.T) {
	testCases := []struct {
		name           string
		username       string
		password       string
		accessToken    string
		expectedMethod string
	}{
		{
			name:           "Username and password provided",
			username:       "user",
			password:       "pass",
			expectedMethod: "userPass",
		},
		{
			name:           "Access token provided",
			accessToken:    "token",
			expectedMethod: "accessToken",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupTestEnv(testEnv{
				username:    tc.username,
				password:    tc.password,
				accessToken: tc.accessToken,
			})
			defer cleanup()
			
			config, err := NewConfig()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			assertEqual(t, tc.expectedMethod, config.Credentials.AuthMethod, 
				"Auth method mismatch")
		})
	}
}

// TestConfigurationParsing verifies environment variables are parsed correctly
func TestConfigurationParsing(t *testing.T) {
	const (
		username    = "test-user"
		password    = "test-password"
		accessToken = "test-token"
	)

	testCases := []struct {
		name      string
		env       testEnv
		checkFunc func(*testing.T, *Config)
	}{
		{
			name: "Username and password parsed correctly",
			env: testEnv{
				username: username,
				password: password,
			},
			checkFunc: func(t *testing.T, config *Config) {
				assertEqual(t, username, config.Credentials.Username, "Username mismatch")
				assertEqual(t, password, config.Credentials.Password, "Password mismatch")
				assertEqual(t, "", config.Credentials.AccessToken, "Access token should be empty")
			},
		},
		{
			name: "Access token parsed correctly",
			env: testEnv{
				accessToken: accessToken,
			},
			checkFunc: func(t *testing.T, config *Config) {
				assertEqual(t, "", config.Credentials.Username, "Username should be empty")
				assertEqual(t, "", config.Credentials.Password, "Password should be empty")
				assertEqual(t, accessToken, config.Credentials.AccessToken, "Access token mismatch")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupTestEnv(tc.env)
			defer cleanup()
			
			config, err := NewConfig()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			tc.checkFunc(t, config)
		})
	}
}

// TestURLValidation verifies URL validation logic
func TestURLValidation(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name: "Valid HTTP URL",
			url:  "http://localhost:8081/artifactory",
		},
		{
			name: "Valid HTTPS URL",
			url:  "https://artifactory.example.com/artifactory",
		},
		{
			name:        "Malformed URL with syntax error",
			url:         "http://[::1]:namedport",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := testEnv{
				username:  "user",
				password:  "pass",
				scrapeURI: tc.url,
			}
			
			cleanup := setupTestEnv(env)
			defer cleanup()
			
			_, err := NewConfig()
			
			assertError(t, err, tc.expectError, fmt.Sprintf("URL: %s", tc.url))
		})
	}
}

// TestOptionalMetricsValidation verifies optional metrics validation logic
func TestOptionalMetricsValidation(t *testing.T) {
	testCases := []struct {
		name            string
		optionalMetrics []string
		expectError     bool
		expectedConfig  OptionalMetrics
	}{
		{
			name:            "No optional metrics",
			optionalMetrics: []string{},
			expectError:     false,
			expectedConfig:  OptionalMetrics{},
		},
		{
			name:            "Single valid optional metric - artifacts",
			optionalMetrics: []string{"artifacts"},
			expectError:     false,
			expectedConfig:  OptionalMetrics{Artifacts: true},
		},
		{
			name:            "Multiple valid optional metrics",
			optionalMetrics: []string{"artifacts", "federation_status"},
			expectError:     false,
			expectedConfig:  OptionalMetrics{
				Artifacts:        true,
				FederationStatus: true,
			},
		},
		{
			name:            "Invalid optional metric",
			optionalMetrics: []string{"non_existent_metric"},
			expectError:     true,
			expectedConfig:  OptionalMetrics{},
		},
		{
			name:            "Mixed valid and invalid metrics",
			optionalMetrics: []string{"artifacts", "non_existent_metric"},
			expectError:     true,
			expectedConfig:  OptionalMetrics{},
		},
		{
			name:            "All valid optional metrics",
			optionalMetrics: []string{
				"artifacts", 
				"replication_status", 
				"federation_status", 
				"open_metrics", 
				"access_federation_validate",
			},
			expectError:     false,
			expectedConfig:  OptionalMetrics{
				Artifacts:               true,
				ReplicationStatus:       true,
				FederationStatus:        true,
				OpenMetrics:             true,
				AccessFederationValidate: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup basic environment to avoid auth errors
			cleanup := setupTestEnv(testEnv{
				username: "user",
				password: "pass",
			})
			defer cleanup()
			
			// Mock the optional metrics flag
			restoreMetrics := mockOptionalMetrics(tc.optionalMetrics)
			defer restoreMetrics()
			
			// For access_federation_validate test, we need to set the target
			if contains(tc.optionalMetrics, "access_federation_validate") {
				os.Setenv("ACCESS_FEDERATION_TARGET", "http://example.com")
				defer os.Unsetenv("ACCESS_FEDERATION_TARGET")
			}
			
			config, err := NewConfig()
			
			// Check error expectation
			contextMsg := fmt.Sprintf("optional metrics: [%s]", 
				strings.Join(tc.optionalMetrics, ", "))
			assertError(t, err, tc.expectError, contextMsg)
			
			// If no error, verify the config
			if err == nil {
				assertEqual(t, tc.expectedConfig, config.OptionalMetrics, 
					"Optional metrics configuration mismatch")
			}
		})
	}
}

// contains checks if a slice contains a string value
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
