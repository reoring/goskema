#!/bin/bash

# Kubernetes CRD Validator サンプルテストスクリプト

set -e

echo "🎯 goskema Kubernetes CRD Validator Demo"
echo "========================================"
echo

# Build the validator
echo "🔨 Building validator..."
go build -o validator .
echo "✅ Build completed"
echo

# Run the demo
echo "🎪 Running full demo..."
./validator demo
echo

echo "📝 Individual command examples:"
echo "------------------------------"

# Test valid widget
echo "✅ Testing valid widget:"
./validator validate valid-widget.yaml
echo

# Test invalid widget (this should fail)
echo "❌ Testing invalid widget (should fail):"
./validator validate invalid-widget.yaml || echo "✅ Expected failure - validation working correctly!"
echo

# Show schema
echo "📋 Showing generated JSON Schema:"
./validator schema
echo

# Test with stdin (simulate kubectl pipe)
echo "📥 Testing stdin input (simulating kubectl output):"
cat valid-widget.yaml | ./validator validate -
echo

echo "🧪 Advanced test cases:"
echo "----------------------"

# Create a test widget with unknown fields
cat > unknown-field-widget.yaml << 'EOF'
apiVersion: example.com/v1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  size: large
  replicas: 5
  config:
    timeout: "1m"
    retries: 3
  unknownField: "this should be rejected due to additionalProperties: false"
EOF

echo "🚫 Testing widget with unknown fields (should fail):"
./validator validate unknown-field-widget.yaml || echo "✅ Unknown fields correctly rejected!"
echo

# Create a test widget with invalid enum value
cat > invalid-enum-widget.yaml << 'EOF'
apiVersion: example.com/v1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  size: "extra-large"  # invalid enum value
  replicas: 3
EOF

echo "🚫 Testing widget with invalid enum (should fail):"
./validator validate invalid-enum-widget.yaml || echo "✅ Invalid enum value correctly rejected!"
echo

# Create a widget with boundary values
cat > boundary-widget.yaml << 'EOF'
apiVersion: example.com/v1
kind: Widget
metadata:
  name: boundary-widget
  namespace: default
spec:
  size: small
  replicas: 1  # minimum value
  config:
    timeout: "1s"
    retries: 0  # minimum value
EOF

echo "✅ Testing widget with boundary values:"
./validator validate boundary-widget.yaml
echo

# Cleanup
rm -f validator unknown-field-widget.yaml invalid-enum-widget.yaml boundary-widget.yaml

echo "🎉 All tests completed!"
echo
echo "🎯 Key Learning Points Demonstrated:"
echo "  ✅ CRD schema import from YAML"
echo "  ✅ Structural schema validation (additionalProperties: false)"
echo "  ✅ Enum value validation"
echo "  ✅ Range validation (min/max values)"
echo "  ✅ Pattern matching for strings"
echo "  ✅ Required field validation"
echo "  ✅ JSON Pointer error paths"
echo "  ✅ YAML input processing"
echo "  ✅ Stdin input support"
echo "  ✅ JSON Schema generation"
echo
echo "🚀 Next Steps:"
echo "  - Try creating your own CRD definition"
echo "  - Experiment with different validation scenarios"
echo "  - Build an AdmissionWebhook using this pattern"
echo "  - Integrate with kubectl as a plugin"
