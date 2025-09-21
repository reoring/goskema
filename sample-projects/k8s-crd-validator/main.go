package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s validate <file|->", os.Args[0])
			os.Exit(1)
		}
		filename := os.Args[2]
		if err := validateWidget(filename); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Validation passed!")

	case "schema":
		if err := showSchema(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to show schema: %v\n", err)
			os.Exit(1)
		}

	case "demo":
		if err := runDemo(); err != nil {
			fmt.Fprintf(os.Stderr, "Demo failed: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`🎯 goskema Kubernetes CRD Validator Sample

Usage: %s <command> [args...]

Commands:
  validate <file|->     Validate a Widget resource from file or stdin
  schema                Show the generated goskema schema for Widget
  demo                  Run validation demo with sample files

Examples:
  %s validate valid-widget.yaml
  %s validate invalid-widget.yaml  
  kubectl get widgets my-widget -o yaml | %s validate -
  %s demo

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func loadCRDSchema() (goskema.Schema[map[string]any], error) {
	// Load CRD definition
	crdData, err := os.ReadFile("widget-crd.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read CRD file: %w", err)
	}

	// Import schema from CRD using kubeopenapi
	schema, diag, err := kubeopenapi.ImportYAMLForCRDKind(
		crdData,
		"Widget",
		kubeopenapi.Options{
			// Enable structural schema compliance for Kubernetes
			Profile: kubeopenapi.ProfileStructuralV1,
			Unknown: kubeopenapi.UnknownStrict,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to import CRD schema: %w", err)
	}

	// Show warnings if any
	if diag.HasWarnings() {
		for _, warning := range diag.Warnings() {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: %s\n", warning)
		}
	}

	return schema, nil
}

func validateWidget(filename string) error {
	ctx := context.Background()

	// Load CRD schema
	schema, err := loadCRDSchema()
	if err != nil {
		return err
	}

	// Read resource file
	var reader io.Reader
	if filename == "-" {
		reader = os.Stdin
		fmt.Fprintf(os.Stderr, "📖 Reading from stdin...\n")
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filename, err)
		}
		defer file.Close()
		reader = file
		fmt.Fprintf(os.Stderr, "📖 Validating %s...\n", filename)
	}

	// Parse and validate using goskema with strict options
	opt := goskema.ParseOpt{
		Strictness: goskema.Strictness{
			OnDuplicateKey: goskema.Error, // Reject duplicate keys
		},
		FailFast: false, // Collect all errors for better UX
	}

	// Read YAML data and convert to JSON for processing
	yamlData, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Parse YAML and convert to map for JSON processing
	var yamlObj map[string]any
	if err := yaml.Unmarshal(yamlData, &yamlObj); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert YAML object to JSON bytes for goskema processing
	jsonData, err := json.Marshal(yamlObj)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	src := goskema.JSONBytes(jsonData)

	result, err := goskema.ParseFrom(ctx, schema, src, opt)
	if err != nil {
		return handleValidationError(err)
	}

	// Show successful result
	fmt.Fprintf(os.Stderr, "🎉 Resource is valid!\n")

	// Pretty print the parsed result
	if metadata, ok := result["metadata"].(map[string]any); ok {
		if name, ok := metadata["name"].(string); ok {
			fmt.Fprintf(os.Stderr, "   📛 Name: %s\n", name)
		}
		if namespace, ok := metadata["namespace"].(string); ok {
			fmt.Fprintf(os.Stderr, "   📁 Namespace: %s\n", namespace)
		}
	}

	if spec, ok := result["spec"].(map[string]any); ok {
		if size, ok := spec["size"].(string); ok {
			fmt.Fprintf(os.Stderr, "   📏 Size: %s\n", size)
		}
		if replicas, ok := spec["replicas"]; ok {
			fmt.Fprintf(os.Stderr, "   🔢 Replicas: %v\n", replicas)
		}
	}

	return nil
}

func handleValidationError(err error) error {
	// Check if it's a goskema validation error with detailed issues
	if issues, ok := goskema.AsIssues(err); ok {
		fmt.Fprintf(os.Stderr, "❌ Validation failed with %d issue(s):\n\n", len(issues))

		for i, issue := range issues {
			fmt.Fprintf(os.Stderr, "  %d. 🚨 %s at %s\n", i+1, issue.Message, issue.Path)
			fmt.Fprintf(os.Stderr, "     Code: %s\n", issue.Code)
			if issue.Hint != "" {
				fmt.Fprintf(os.Stderr, "     Hint: %s\n", issue.Hint)
			}
			if issue.InputFragment != "" {
				fmt.Fprintf(os.Stderr, "     Input: %s\n", issue.InputFragment)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
		return fmt.Errorf("validation failed with %d issue(s)", len(issues))
	}

	return fmt.Errorf("validation error: %w", err)
}

func showSchema() error {
	// Load CRD schema
	schema, err := loadCRDSchema()
	if err != nil {
		return err
	}

	// Generate JSON Schema
	jsonSchema, err := schema.JSONSchema()
	if err != nil {
		return fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	// Pretty print JSON Schema
	fmt.Println("📋 Generated JSON Schema for Widget:")
	fmt.Println()

	// Convert to YAML for readability
	yamlData, err := yaml.Marshal(jsonSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	fmt.Print(string(yamlData))
	return nil
}

func runDemo() error {
	fmt.Println("🎪 Running goskema CRD Validation Demo")
	fmt.Println("=====================================")
	fmt.Println()

	// Test valid widget
	fmt.Println("1️⃣ Testing valid Widget resource:")
	fmt.Println("----------------------------------")
	if err := validateWidget("valid-widget.yaml"); err != nil {
		return fmt.Errorf("valid widget test failed: %w", err)
	}
	fmt.Println()

	// Test invalid widget
	fmt.Println("2️⃣ Testing invalid Widget resource:")
	fmt.Println("------------------------------------")
	if err := validateWidget("invalid-widget.yaml"); err != nil {
		fmt.Fprintf(os.Stderr, "Expected validation failure: %v\n", err)
	}
	fmt.Println()

	// Show schema
	fmt.Println("3️⃣ Generated JSON Schema:")
	fmt.Println("--------------------------")
	if err := showSchema(); err != nil {
		return fmt.Errorf("schema generation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✨ Demo completed!")
	fmt.Println()
	fmt.Println("🎯 Key Learning Points:")
	fmt.Println("  - CRD schema import and validation")
	fmt.Println("  - Structural schema compliance")
	fmt.Println("  - YAML resource processing")
	fmt.Println("  - Detailed error reporting with JSON Pointer paths")
	fmt.Println("  - JSON Schema generation from goskema schemas")

	return nil
}

func init() {
	// Setup logging for better debug experience
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
