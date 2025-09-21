#!/bin/bash

# Config Manager サンプルテストスクリプト

set -e

echo "🎯 goskema Config Manager Demo"
echo "============================"
echo

# Clean up any existing files
rm -f base.yaml development.yaml staging.yaml production.yaml

# Generate template configurations
echo "📝 Generating template configuration files..."
go run . generate --template
echo

# Show generated files
echo "📁 Generated configuration files:"
ls -la *.yaml
echo

# Set some test environment variables
export DB_PASSWORD="test_db_password"
export REDIS_PASSWORD="test_redis_password"
export DB_HOST="test-db.example.com"
export REDIS_HOST="test-redis.example.com"
export TLS_CERT_FILE="/path/to/cert.pem"
export TLS_KEY_FILE="/path/to/key.pem"
export LOG_OUTPUT="file:/var/log/app.log"

echo "🧪 Testing configuration validation..."
echo "------------------------------------"

# Test development environment
echo "✅ Testing development environment:"
go run . validate --env=development
echo

# Test staging environment  
echo "✅ Testing staging environment:"
go run . validate --env=staging
echo

# Test production environment
echo "✅ Testing production environment:"
go run . validate --env=production
echo

echo "👀 Showing configurations..."
echo "---------------------------"

# Show development config (with secrets masked)
echo "🔍 Development configuration (secrets masked):"
go run . show --env=development
echo

# Show production config without masking (for demo)
echo "🔍 Production configuration (secrets visible):"
go run . show --env=production --no-mask
echo

# Show JSON Schema
echo "📋 Configuration JSON Schema:"
echo "----------------------------"
go run . schema
echo

echo "🧪 Testing error scenarios..."
echo "----------------------------"

# Create an invalid configuration
cat > invalid.yaml << 'EOF'
app:
  name: "TestApp"
  version: "1.0.0"
  port: 99999  # Invalid port number
  
database:
  host: "localhost"
  # Missing required fields: database, username
  
logging:
  level: "invalid_level"  # Invalid log level
EOF

echo "❌ Testing invalid configuration (should fail):"
if go run . validate --env=development 2>/dev/null; then
    echo "❌ Validation should have failed!"
else
    echo "✅ Invalid configuration correctly rejected!"
fi
echo

# Test with missing environment variables
echo "❌ Testing missing environment variables:"
unset DB_HOST REDIS_HOST TLS_CERT_FILE TLS_KEY_FILE

# This should work because base.yaml has defaults
echo "✅ Development still works with missing env vars (has defaults):"
go run . validate --env=development
echo

# But production might have issues
echo "⚠️  Production with missing required env vars:"
if go run . validate --env=production 2>/dev/null; then
    echo "⚠️  Production validation passed (some values may be empty)"
else
    echo "❌ Production validation failed due to missing env vars"
fi
echo

# Restore environment variables for final tests
export DB_HOST="test-db.example.com"
export REDIS_HOST="test-redis.example.com"
export TLS_CERT_FILE="/path/to/cert.pem"
export TLS_KEY_FILE="/path/to/key.pem"

echo "🔄 Testing environment variable expansion..."
echo "------------------------------------------"

# Create a test config with various env var patterns
cat > env-test.yaml << 'EOF'
app:
  name: "${APP_NAME:-DefaultApp}"
  environment: "${ENVIRONMENT}"
  port: 8080

database:
  host: "${DB_HOST}"
  password: "${DB_PASSWORD:-default_password}"
  
features:
  debugging: true
EOF

# Set some test variables
export APP_NAME="MyTestApp"
export ENVIRONMENT="test"

echo "🔍 Testing environment variable expansion:"
echo "Environment variables:"
echo "  APP_NAME=$APP_NAME"
echo "  ENVIRONMENT=$ENVIRONMENT"
echo "  DB_HOST=$DB_HOST"
echo "  DB_PASSWORD=$DB_PASSWORD"
echo

# Test the expansion
mv base.yaml base.yaml.backup
cp env-test.yaml base.yaml

echo "Configuration after expansion:"
go run . show --env=development --no-mask
echo

# Restore original base.yaml
mv base.yaml.backup base.yaml

# Clean up test files
rm -f invalid.yaml env-test.yaml

echo "✨ All tests completed!"
echo
echo "🎯 Key Learning Points Demonstrated:"
echo "  ✅ Hierarchical configuration (base + environment overrides)"
echo "  ✅ Environment variable expansion with defaults"
echo "  ✅ Comprehensive validation using goskema DSL"
echo "  ✅ Secret masking for security"
echo "  ✅ Error handling and validation"
echo "  ✅ JSON Schema generation"
echo "  ✅ Template configuration generation"
echo
echo "🚀 Next Steps:"
echo "  - Customize the configuration structure for your needs"  
echo "  - Add more validation rules using goskema"
echo "  - Implement hot-reload functionality"
echo "  - Integrate with your application"
echo "  - Add support for more configuration formats"

# Clean up generated files
echo
read -p "🧹 Clean up generated files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f base.yaml development.yaml staging.yaml production.yaml
    echo "🗑️  Cleaned up generated files"
fi
