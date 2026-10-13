package config

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// GeneratePassword creates a memorable password from two random words joined by a hyphen.
func GeneratePassword() string {
	w1 := wordList[rand.Intn(len(wordList))]
	w2 := wordList[rand.Intn(len(wordList))]
	return w1 + "-" + w2
}

// Config holds all application configuration
type Config struct {
	// Server configuration
	Password     string
	Port         string
	DownloadsDir string

	// ngrok configuration
	NgrokAuthToken string

	// AWS CloudFront configuration
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	CloudFrontDistID   string

	// CFOriginSecret is injected as a custom origin header on every request
	// CloudFront forwards, so the app can tell CloudFront traffic apart from
	// someone hitting the ngrok URL directly. Generated and persisted on first
	// run; a stable value avoids a propagation gap on every restart.
	CFOriginSecret string

	// Access control. TrustedIPs comes from .env and is always allowed;
	// AllowedIPs is populated from the --allow-ips flag by the caller.
	TrustedIPs string
	AllowedIPs string

	// envFile is the resolved path of the .env file in use (or where one
	// would be created), so writes go to the same file that was loaded.
	envFile string
}

// locateEnv finds the .env file: the working directory takes precedence
// (preserves `go run` and in-place usage), then the directory containing the
// executable with symlinks resolved, so a symlinked binary still finds its
// project config. Returns the .env path (which may not exist yet) and the
// base directory used to resolve relative paths like DOWNLOADS_DIR.
func locateEnv() (envFile, baseDir string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	if _, err := os.Stat(filepath.Join(cwd, ".env")); err == nil {
		return filepath.Join(cwd, ".env"), cwd
	}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		exeDir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(exeDir, ".env")); err == nil {
			return filepath.Join(exeDir, ".env"), exeDir
		}
	}

	// No .env anywhere; fall back to the working directory.
	return filepath.Join(cwd, ".env"), cwd
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	envFile, baseDir := locateEnv()

	// Load .env file if it exists (ignore error if file doesn't exist)
	godotenv.Load(envFile)

	cfg := &Config{
		envFile: envFile,
		Password:           os.Getenv("PASSWORD"),
		Port:               getEnvOrDefault("PORT", "8081"),
		NgrokAuthToken:     os.Getenv("NGROK_AUTHTOKEN"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:          getEnvOrDefault("AWS_REGION", "us-east-1"),
		CloudFrontDistID:   os.Getenv("CLOUDFRONT_DISTRIBUTION_ID"),
		CFOriginSecret:     os.Getenv("CLOUDFRONT_ORIGIN_SECRET"),
		TrustedIPs:         os.Getenv("TRUSTED_IPS"),
	}

	// Generate password if not provided
	if cfg.Password == "" {
		cfg.Password = GeneratePassword()
	}

	// Generate the CloudFront origin secret once and persist it. Rotating it
	// on every start would leave the app unreachable through CloudFront until
	// the new value propagates (~5 minutes).
	if cfg.CFOriginSecret == "" {
		secret, err := generateSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate CloudFront origin secret: %w", err)
		}
		cfg.CFOriginSecret = secret
		if err := cfg.SaveOriginSecret(secret); err != nil {
			return nil, fmt.Errorf("failed to save CloudFront origin secret: %w", err)
		}
	}

	// Resolve downloads directory to an absolute path; relative paths are
	// resolved against the directory the .env file was found in.
	downloadsDir := getEnvOrDefault("DOWNLOADS_DIR", "downloads")
	if !filepath.IsAbs(downloadsDir) {
		downloadsDir = filepath.Join(baseDir, downloadsDir)
	}
	downloadsDir, err := filepath.Abs(downloadsDir)
	if err != nil {
		return nil, fmt.Errorf("invalid DOWNLOADS_DIR: %w", err)
	}
	cfg.DownloadsDir = downloadsDir

	return cfg, nil
}

// AllowlistEntries returns the merged, de-duplicated set of IPs and CIDRs from
// TRUSTED_IPS and --allow-ips. An empty result means no IP restriction.
func (c *Config) AllowlistEntries() []string {
	seen := make(map[string]bool)
	var entries []string

	for _, list := range []string{c.TrustedIPs, c.AllowedIPs} {
		for _, entry := range strings.Split(list, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" || seen[entry] {
				continue
			}
			seen[entry] = true
			entries = append(entries, entry)
		}
	}

	return entries
}

// HasAWSCredentials checks if AWS credentials are configured
func (c *Config) HasAWSCredentials() bool {
	return c.AWSAccessKeyID != "" && c.AWSSecretAccessKey != ""
}

// HasCloudFrontConfig checks if CloudFront is configured
func (c *Config) HasCloudFrontConfig() bool {
	return c.HasAWSCredentials() && c.CloudFrontDistID != ""
}

// generateSecret returns a cryptographically random, URL-safe token.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// upsertEnvVar sets key=value in the .env file, replacing an existing entry or
// appending a new one. Creates the file if it doesn't exist.
func upsertEnvVar(envFile, key, value string) error {
	content, err := os.ReadFile(envFile)
	if err != nil {
		content = []byte("")
	}

	prefix := key + "="
	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + value
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, prefix+value)
	}

	return os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0644)
}

// removeEnvVar deletes any entry for key from the .env file.
func removeEnvVar(envFile, key string) error {
	content, err := os.ReadFile(envFile)
	if err != nil {
		// If file doesn't exist, nothing to remove
		return nil
	}

	prefix := key + "="
	var newLines []string
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(line, prefix) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644)
}

// SaveDistributionID saves the CloudFront distribution ID to .env file
func (c *Config) SaveDistributionID(distributionID string) error {
	c.CloudFrontDistID = distributionID
	return upsertEnvVar(c.envFile, "CLOUDFRONT_DISTRIBUTION_ID", distributionID)
}

// RemoveDistributionID removes the CloudFront distribution ID from .env file
func (c *Config) RemoveDistributionID() error {
	c.CloudFrontDistID = ""
	return removeEnvVar(c.envFile, "CLOUDFRONT_DISTRIBUTION_ID")
}

// SaveOriginSecret saves the CloudFront origin secret to .env file
func (c *Config) SaveOriginSecret(secret string) error {
	c.CFOriginSecret = secret
	return upsertEnvVar(c.envFile, "CLOUDFRONT_ORIGIN_SECRET", secret)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
