package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	App      AppConfig      `json:"app"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Logging  LoggingConfig  `json:"logging"`
	Features FeaturesConfig `json:"features"`
}

type AppConfig struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Port        int               `json:"port"`
	Host        string            `json:"host"`
	TLS         TLSConfig         `json:"tls"`
	Cors        CorsConfig        `json:"cors"`
	Metadata    map[string]string `json:"metadata"`
}

type TLSConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

type CorsConfig struct {
	Enabled bool     `json:"enabled"`
	Origins []string `json:"origins"`
}

type DatabaseConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Database     string `json:"database"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	MaxConns     int    `json:"maxConns"`
	MaxIdleConns int    `json:"maxIdleConns"`
	SSLMode      string `json:"sslMode"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database int    `json:"database"`
	Password string `json:"password"`
	PoolSize int    `json:"poolSize"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

type FeaturesConfig struct {
	Analytics bool `json:"analytics"`
	Debugging bool `json:"debugging"`
}

// ConfigManager handles configuration loading and validation
type ConfigManager struct {
	schema goskema.Schema[Config]
}

func NewConfigManager() *ConfigManager {
	// Define comprehensive schema using goskema DSL
	schema := g.ObjectOf[Config]().
		Field("app", g.ObjectOf[AppConfig]().
			Field("name", g.StringOf[string]()).Required().
			Field("version", g.StringOf[string]()).Required().
			Field("environment", g.StringOf[string]()).Default("development").
			Field("port", g.NumberOf[int]()).Default(8080).
			Field("host", g.StringOf[string]()).Default("0.0.0.0").
			Field("tls", g.ObjectOf[TLSConfig]().
				Field("enabled", g.BoolOf[bool]()).Default(false).
				Field("certFile", g.StringOf[string]()).Default("").
				Field("keyFile", g.StringOf[string]()).Default("").
				UnknownStrict().
				MustBind()).Default(TLSConfig{}).
			Field("cors", g.ObjectOf[CorsConfig]().
				Field("enabled", g.BoolOf[bool]()).Default(true).
				Field("origins", g.Array[string](g.StringOf[string]())).Default([]string{"*"}).
				UnknownStrict().
				MustBind()).Default(CorsConfig{Enabled: true, Origins: []string{"*"}}).
			Field("metadata", g.MapOf[string](g.StringOf[string]())).Default(map[string]string{}).
			UnknownStrict().
			MustBind()).Required().
		Field("database", g.ObjectOf[DatabaseConfig]().
			Field("host", g.StringOf[string]()).Required().
			Field("port", g.NumberOf[int]()).Default(5432).
			Field("database", g.StringOf[string]()).Required().
			Field("username", g.StringOf[string]()).Required().
			Field("password", g.StringOf[string]()).Default("").
			Field("maxConns", g.NumberOf[int]()).Default(10).
			Field("maxIdleConns", g.NumberOf[int]()).Default(5).
			Field("sslMode", g.StringOf[string]()).Default("prefer").
			UnknownStrict().
			MustBind()).Required().
		Field("redis", g.ObjectOf[RedisConfig]().
			Field("host", g.StringOf[string]()).Default("localhost").
			Field("port", g.NumberOf[int]()).Default(6379).
			Field("database", g.NumberOf[int]()).Default(0).
			Field("password", g.StringOf[string]()).Default("").
			Field("poolSize", g.NumberOf[int]()).Default(10).
			UnknownStrict().
			MustBind()).Required().
		Field("logging", g.ObjectOf[LoggingConfig]().
			Field("level", g.StringOf[string]()).Default("info").
			Field("format", g.StringOf[string]()).Default("json").
			Field("output", g.StringOf[string]()).Default("stdout").
			UnknownStrict().
			MustBind()).Required().
		Field("features", g.ObjectOf[FeaturesConfig]().
			Field("analytics", g.BoolOf[bool]()).Default(true).
			Field("debugging", g.BoolOf[bool]()).Default(false).
			UnknownStrict().
			MustBind()).Required().
		UnknownStrict().
		MustBind()

	return &ConfigManager{
		schema: schema,
	}
}

func (cm *ConfigManager) LoadConfig(env string) (Config, error) {
	ctx := context.Background()

	// Load base configuration
	baseData, err := cm.loadFile("base.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load base config: %w", err)
	}

	// Expand environment variables in base config
	baseData = cm.expandEnvVars(baseData)

	// Parse base config
	baseConfig, err := goskema.ParseFrom(ctx, cm.schema, goskema.YAMLBytes(baseData))
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load environment-specific configuration if it exists
	envFile := fmt.Sprintf("%s.yaml", env)
	if cm.fileExists(envFile) {
		envData, err := cm.loadFile(envFile)
		if err != nil {
			return Config{}, fmt.Errorf("failed to load %s config: %w", env, err)
		}

		// Expand environment variables in env config
		envData = cm.expandEnvVars(envData)

		// Parse environment config
		envConfig, err := goskema.ParseFrom(ctx, cm.schema, goskema.YAMLBytes(envData))
		if err != nil {
			return Config{}, fmt.Errorf("failed to parse %s config: %w", env, err)
		}

		// Merge configurations (env config overrides base)
		return cm.mergeConfigs(baseConfig, envConfig), nil
	}

	return baseConfig, nil
}

func (cm *ConfigManager) ValidateConfig(env string, streaming bool) error {
	ctx := context.Background()

	config, err := cm.LoadConfig(env)
	if err != nil {
		return err
	}

	// Additional validation logic
	if config.App.Port < 1 || config.App.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", config.App.Port)
	}

	if config.App.TLS.Enabled && (config.App.TLS.CertFile == "" || config.App.TLS.KeyFile == "") {
		return fmt.Errorf("TLS enabled but cert/key files not specified")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	fmt.Printf("✅ Configuration for environment '%s' is valid!\n", env)
	return nil
}

func (cm *ConfigManager) ShowConfig(env string, maskSecrets bool) error {
	config, err := cm.LoadConfig(env)
	if err != nil {
		return err
	}

	if maskSecrets {
		config = cm.maskSecrets(config)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Printf("📋 Configuration for environment: %s\n", env)
	fmt.Println("=" + strings.Repeat("=", len(env)+25))
	fmt.Print(string(data))

	return nil
}

func (cm *ConfigManager) GenerateTemplate() error {
	// Generate template configurations
	templates := map[string]string{
		"base.yaml": `# Base configuration (common settings)
app:
  name: "MyWebApp"
  version: "1.0.0"
  host: "0.0.0.0"
  port: 8080
  tls:
    enabled: false
  cors:
    enabled: true
    origins: ["*"]
  metadata:
    author: "Your Name"
    description: "Web application"

database:
  host: "localhost"
  port: 5432
  database: "myapp"
  username: "postgres"
  maxConns: 10
  maxIdleConns: 5
  sslMode: "prefer"

redis:
  host: "localhost"
  port: 6379
  database: 0
  poolSize: 10

logging:
  level: "info"
  format: "json"
  output: "stdout"

features:
  analytics: true
  debugging: false
`,
		"development.yaml": `# Development environment overrides
app:
  environment: "development"
  port: 3000

database:
  password: "${DB_PASSWORD:-dev_password}"
  sslMode: "disable"

redis:
  password: "${REDIS_PASSWORD:-}"

logging:
  level: "debug"

features:
  debugging: true
`,
		"staging.yaml": `# Staging environment overrides
app:
  environment: "staging"
  port: 8080
  cors:
    origins: ["https://staging.example.com"]

database:
  host: "${DB_HOST:-staging-db.example.com}"
  password: "${DB_PASSWORD}"
  sslMode: "require"

redis:
  host: "${REDIS_HOST:-staging-redis.example.com}"
  password: "${REDIS_PASSWORD}"

logging:
  level: "info"
`,
		"production.yaml": `# Production environment overrides
app:
  environment: "production"
  port: 80
  tls:
    enabled: true
    certFile: "${TLS_CERT_FILE}"
    keyFile: "${TLS_KEY_FILE}"
  cors:
    origins: ["https://example.com", "https://app.example.com"]

database:
  host: "${DB_HOST}"
  password: "${DB_PASSWORD}"
  maxConns: 50
  maxIdleConns: 10
  sslMode: "require"

redis:
  host: "${REDIS_HOST}"
  password: "${REDIS_PASSWORD}"
  poolSize: 50

logging:
  level: "warn"
  output: "${LOG_OUTPUT:-stdout}"

features:
  debugging: false
`,
	}

	for filename, content := range templates {
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		fmt.Printf("📝 Generated %s\n", filename)
	}

	fmt.Println("✅ Template configuration files generated!")
	fmt.Println("\n📖 Next steps:")
	fmt.Println("1. Edit the configuration files as needed")
	fmt.Println("2. Set required environment variables")
	fmt.Println("3. Validate with: go run . validate --env=development")

	return nil
}

func (cm *ConfigManager) loadFile(filename string) ([]byte, error) {
	if !cm.fileExists(filename) {
		return nil, fmt.Errorf("file %s does not exist", filename)
	}
	return os.ReadFile(filename)
}

func (cm *ConfigManager) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (cm *ConfigManager) expandEnvVars(data []byte) []byte {
	content := string(data)

	// Match ${VAR} and ${VAR:-default} patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// Remove ${ and }
		varExpr := match[2 : len(match)-1]

		// Check for default value syntax
		if strings.Contains(varExpr, ":-") {
			parts := strings.SplitN(varExpr, ":-", 2)
			varName := parts[0]
			defaultValue := parts[1]

			if value := os.Getenv(varName); value != "" {
				return value
			}
			return defaultValue
		}

		// Simple variable substitution
		return os.Getenv(varExpr)
	})

	return []byte(result)
}

func (cm *ConfigManager) mergeConfigs(base, override Config) Config {
	// Simple merge - in a real implementation, you might want more sophisticated merging
	result := base

	// Override non-zero values from override config
	if override.App.Environment != "" {
		result.App.Environment = override.App.Environment
	}
	if override.App.Port != 0 {
		result.App.Port = override.App.Port
	}
	if override.Database.Host != "" {
		result.Database.Host = override.Database.Host
	}
	if override.Database.Password != "" {
		result.Database.Password = override.Database.Password
	}
	if override.Redis.Host != "" {
		result.Redis.Host = override.Redis.Host
	}
	if override.Redis.Password != "" {
		result.Redis.Password = override.Redis.Password
	}
	if override.Logging.Level != "" {
		result.Logging.Level = override.Logging.Level
	}
	// ... (simplified merge logic)

	return result
}

func (cm *ConfigManager) maskSecrets(config Config) Config {
	masked := config

	// Mask sensitive information
	if masked.Database.Password != "" {
		masked.Database.Password = "***masked***"
	}
	if masked.Redis.Password != "" {
		masked.Redis.Password = "***masked***"
	}
	if masked.App.TLS.KeyFile != "" {
		masked.App.TLS.KeyFile = "***masked***"
	}

	return masked
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cm := NewConfigManager()
	command := os.Args[1]

	switch command {
	case "validate":
		env := getEnvFlag()
		streaming := getStreamingFlag()
		if err := cm.ValidateConfig(env, streaming); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
			os.Exit(1)
		}

	case "show":
		env := getEnvFlag()
		maskSecrets := !getBoolFlag("--no-mask")
		if err := cm.ShowConfig(env, maskSecrets); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Show failed: %v\n", err)
			os.Exit(1)
		}

	case "generate":
		if getBoolFlag("--template") {
			if err := cm.GenerateTemplate(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Generate failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "❌ Use --template flag to generate template files\n")
			os.Exit(1)
		}

	case "schema":
		schema, err := cm.schema.JSONSchema()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Schema generation failed: %v\n", err)
			os.Exit(1)
		}

		data, err := yaml.Marshal(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Schema marshal failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("📋 Configuration JSON Schema:")
		fmt.Print(string(data))

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`🎯 goskema Config Manager Sample

Usage: %s <command> [flags...]

Commands:
  validate [--env=<env>] [--streaming]  Validate configuration for environment
  show [--env=<env>] [--no-mask]        Show configuration (default: mask secrets)
  generate --template                   Generate template configuration files
  schema                                Show JSON Schema for configuration

Flags:
  --env=<environment>      Environment (default: development)
  --streaming              Use streaming processing for large files
  --no-mask               Don't mask sensitive information
  --template              Generate template files

Examples:
  %s validate --env=development
  %s validate --env=production --streaming  
  %s show --env=staging
  %s show --env=production --no-mask
  %s generate --template
  %s schema

Environment Files:
  base.yaml               Base configuration (required)
  <environment>.yaml      Environment-specific overrides (optional)

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func getEnvFlag() string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--env=") {
			return strings.TrimPrefix(arg, "--env=")
		}
	}
	return "development"
}

func getStreamingFlag() bool {
	return getBoolFlag("--streaming")
}

func getBoolFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
