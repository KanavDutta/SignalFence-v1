package signalfence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Check defaults
	if config.Defaults.Capacity != 100 {
		t.Errorf("Defaults.Capacity = %d, want 100", config.Defaults.Capacity)
	}
	if config.Defaults.RefillRate != 10.0 {
		t.Errorf("Defaults.RefillRate = %f, want 10.0", config.Defaults.RefillRate)
	}
	if !config.Defaults.Enabled {
		t.Error("Defaults.Enabled = false, want true")
	}

	// Check other fields
	if config.KeyExtractor != "ip" {
		t.Errorf("KeyExtractor = %s, want ip", config.KeyExtractor)
	}
	if config.CleanupAge != "1h" {
		t.Errorf("CleanupAge = %s, want 1h", config.CleanupAge)
	}
	if config.Policies == nil {
		t.Error("Policies map should be initialized")
	}
}

func TestPolicyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  PolicyConfig
		wantErr bool
		errType error
	}{
		{
			name: "valid policy",
			policy: PolicyConfig{
				Capacity:   100,
				RefillRate: 10.0,
				Enabled:    true,
			},
			wantErr: false,
		},
		{
			name: "zero capacity",
			policy: PolicyConfig{
				Capacity:   0,
				RefillRate: 10.0,
				Enabled:    true,
			},
			wantErr: true,
			errType: ErrNegativeCapacity,
		},
		{
			name: "negative capacity",
			policy: PolicyConfig{
				Capacity:   -10,
				RefillRate: 10.0,
				Enabled:    true,
			},
			wantErr: true,
			errType: ErrNegativeCapacity,
		},
		{
			name: "zero refill rate",
			policy: PolicyConfig{
				Capacity:   100,
				RefillRate: 0,
				Enabled:    true,
			},
			wantErr: true,
			errType: ErrNegativeRefillRate,
		},
		{
			name: "negative refill rate",
			policy: PolicyConfig{
				Capacity:   100,
				RefillRate: -5.0,
				Enabled:    true,
			},
			wantErr: true,
			errType: ErrNegativeRefillRate,
		},
		{
			name: "fractional refill rate",
			policy: PolicyConfig{
				Capacity:   10,
				RefillRate: 0.5,
				Enabled:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
				if err != tt.errType {
					t.Errorf("Validate() error = %v, want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  NewConfig(),
			wantErr: false,
		},
		{
			name: "invalid defaults",
			config: &Config{
				Defaults: PolicyConfig{
					Capacity:   0,
					RefillRate: 10.0,
					Enabled:    true,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid route policy",
			config: &Config{
				Defaults: PolicyConfig{
					Capacity:   100,
					RefillRate: 10.0,
					Enabled:    true,
				},
				Policies: map[string]PolicyConfig{
					"/api/test": {
						Capacity:   -10,
						RefillRate: 5.0,
						Enabled:    true,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_GetPolicy(t *testing.T) {
	config := NewConfig()

	// Add specific policy for /api/login
	loginPolicy := PolicyConfig{
		Capacity:   5,
		RefillRate: 0.083,
		Enabled:    true,
	}
	config.Policies["/api/login"] = loginPolicy

	tests := []struct {
		name   string
		route  string
		want   PolicyConfig
	}{
		{
			name:  "route with specific policy",
			route: "/api/login",
			want:  loginPolicy,
		},
		{
			name:  "route without specific policy (uses defaults)",
			route: "/api/search",
			want:  config.Defaults,
		},
		{
			name:  "empty route (uses defaults)",
			route: "",
			want:  config.Defaults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.GetPolicy(tt.route)
			if got.Capacity != tt.want.Capacity {
				t.Errorf("GetPolicy(%s).Capacity = %d, want %d", tt.route, got.Capacity, tt.want.Capacity)
			}
			if got.RefillRate != tt.want.RefillRate {
				t.Errorf("GetPolicy(%s).RefillRate = %f, want %f", tt.route, got.RefillRate, tt.want.RefillRate)
			}
		})
	}
}

func TestConfig_SetPolicy(t *testing.T) {
	config := NewConfig()

	// Set valid policy
	validPolicy := PolicyConfig{
		Capacity:   50,
		RefillRate: 5.0,
		Enabled:    true,
	}
	err := config.SetPolicy("/api/test", validPolicy)
	if err != nil {
		t.Errorf("SetPolicy() unexpected error: %v", err)
	}

	// Verify policy was set
	got := config.GetPolicy("/api/test")
	if got.Capacity != validPolicy.Capacity {
		t.Errorf("policy not set correctly: Capacity = %d, want %d", got.Capacity, validPolicy.Capacity)
	}

	// Try to set invalid policy
	invalidPolicy := PolicyConfig{
		Capacity:   0,
		RefillRate: 5.0,
		Enabled:    true,
	}
	err = config.SetPolicy("/api/invalid", invalidPolicy)
	if err == nil {
		t.Error("SetPolicy() expected error for invalid policy, got nil")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Test 1: Valid config file
	validConfig := `
defaults:
  capacity: 100
  refill_rate: 10.0
  enabled: true

policies:
  "/api/login":
    capacity: 5
    refill_rate: 0.083
    enabled: true

  "/api/search":
    capacity: 200
    refill_rate: 20.0
    enabled: true

key_extractor: "header:X-API-Key"
cleanup_age: "30m"
`
	validPath := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadConfigFromFile(validPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile() unexpected error: %v", err)
	}

	// Verify loaded values
	if config.Defaults.Capacity != 100 {
		t.Errorf("Defaults.Capacity = %d, want 100", config.Defaults.Capacity)
	}
	if config.KeyExtractor != "header:X-API-Key" {
		t.Errorf("KeyExtractor = %s, want header:X-API-Key", config.KeyExtractor)
	}
	if config.CleanupAge != "30m" {
		t.Errorf("CleanupAge = %s, want 30m", config.CleanupAge)
	}

	// Verify policies
	loginPolicy := config.GetPolicy("/api/login")
	if loginPolicy.Capacity != 5 {
		t.Errorf("/api/login Capacity = %d, want 5", loginPolicy.Capacity)
	}

	searchPolicy := config.GetPolicy("/api/search")
	if searchPolicy.Capacity != 200 {
		t.Errorf("/api/search Capacity = %d, want 200", searchPolicy.Capacity)
	}

	// Test 2: Invalid YAML
	invalidYAML := `
defaults:
  capacity: 100
  invalid yaml here {[
`
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadConfigFromFile(invalidPath)
	if err == nil {
		t.Error("LoadConfigFromFile() expected error for invalid YAML, got nil")
	}

	// Test 3: Invalid config (negative capacity)
	invalidConfig := `
defaults:
  capacity: -10
  refill_rate: 10.0
  enabled: true
`
	invalidConfigPath := filepath.Join(tmpDir, "invalid_config.yaml")
	if err := os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadConfigFromFile(invalidConfigPath)
	if err == nil {
		t.Error("LoadConfigFromFile() expected error for invalid config, got nil")
	}

	// Test 4: File not found
	_, err = LoadConfigFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("LoadConfigFromFile() expected error for nonexistent file, got nil")
	}
}

func TestLoadConfigFromFile_Defaults(t *testing.T) {
	// Create config with minimal fields (should apply defaults)
	tmpDir := t.TempDir()

	minimalConfig := `
defaults:
  capacity: 50
  refill_rate: 5.0
  enabled: true
`
	minimalPath := filepath.Join(tmpDir, "minimal.yaml")
	if err := os.WriteFile(minimalPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadConfigFromFile(minimalPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile() unexpected error: %v", err)
	}

	// Should apply default key extractor
	if config.KeyExtractor != "ip" {
		t.Errorf("KeyExtractor = %s, want ip (default)", config.KeyExtractor)
	}

	// Should apply default cleanup age
	if config.CleanupAge != "1h" {
		t.Errorf("CleanupAge = %s, want 1h (default)", config.CleanupAge)
	}

	// Should initialize empty policies map
	if config.Policies == nil {
		t.Error("Policies map should be initialized")
	}
}

func TestPolicyConfig_ToBucketConfig(t *testing.T) {
	policy := PolicyConfig{
		Capacity:   100,
		RefillRate: 10.0,
		Enabled:    true,
	}

	bucketConfig := policy.ToBucketConfig()

	if bucketConfig.Capacity != policy.Capacity {
		t.Errorf("BucketConfig.Capacity = %d, want %d", bucketConfig.Capacity, policy.Capacity)
	}
	if bucketConfig.RefillRate != policy.RefillRate {
		t.Errorf("BucketConfig.RefillRate = %f, want %f", bucketConfig.RefillRate, policy.RefillRate)
	}
}
