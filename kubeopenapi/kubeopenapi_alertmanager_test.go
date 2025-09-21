package kubeopenapi_test

import (
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Alertmanager_FromBundle_PropertyNames_Smoke(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Alertmanager", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	// minimal spec should pass
	base := []byte(`{
		"apiVersion": "monitoring.coreos.com/v1",
		"kind": "Alertmanager",
		"spec": {}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(base)); err != nil {
		t.Fatalf("base minimal spec should pass: %v", err)
	}
	// NOTE: Real Alertmanager spec may not define propertyNames on a top-level map succinctly.
	// This smoke test ensures at least the import works and base passes. Detailed propertyNames
	// assertions are covered in unit tests.
}

func TestImport_Alertmanager_Alias_Receivers_ListMapKeys_ByName(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_alias_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
		"receivers": [
			{"name": "a"},
			{"name": "b"}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected receivers unique by name: %v", err)
	}
	dup := []byte(`{
		"receivers": [
			{"name": "a"},
			{"name": "a"}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(dup)); err == nil {
		t.Fatalf("expected duplicate_item for receivers with same name")
	}
}

func TestImport_Alertmanager_Routes_Nested_ListMapKeys_ByReceiver(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_routes_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
		"route": {
			"routes": [
				{
					"receiver": "a",
					"routes": [
						{"receiver": "a-child"}
					]
				}
			]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected nested routes unique by receiver: %v", err)
	}
	dupTop := []byte(`{
		"route": {
			"routes": [
				{"receiver": "a"},
				{"receiver": "a"}
			]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(dupTop)); err == nil {
		t.Fatalf("expected duplicate_item for top-level routes receiver")
	}
	dupChild := []byte(`{
		"route": {
			"routes": [
				{
					"receiver": "a",
					"routes": [
						{"receiver": "x"},
						{"receiver": "x"}
					]
				}
			]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(dupChild)); err == nil {
		t.Fatalf("expected duplicate_item for child routes receiver")
	}
}

func TestImport_Alertmanager_Matchers_StringVsObject(t *testing.T) {
	ctx := context.Background()
	// string form
	b1, err := os.ReadFile("testdata/alertmanager_matchers_string_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s1, _, err := kubeopenapi.Import(b1, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if _, err := goskema.ParseFrom(ctx, s1, goskema.JSONBytes([]byte(`{
		"matchers": ["env=prod", "app=web"]
	}`))); err != nil {
		t.Fatalf("expected string matchers accepted: %v", err)
	}
	// object form
	b2, err := os.ReadFile("testdata/alertmanager_matchers_object_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s2, _, err := kubeopenapi.Import(b2, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if _, err := goskema.ParseFrom(ctx, s2, goskema.JSONBytes([]byte(`{
		"matchers": [
			{"name": "env", "value": "prod", "regex": false}
		]
	}`))); err != nil {
		t.Fatalf("expected object matchers accepted: %v", err)
	}
}

func TestImport_AlertmanagerConfig_FromBundle_MinimalOK(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "AlertmanagerConfig", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	js := []byte(`{
		"apiVersion": "monitoring.coreos.com/v1alpha1",
		"kind": "AlertmanagerConfig",
		"spec": {}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("minimal AlertmanagerConfig object should pass: %v", err)
	}
}

func TestImport_AlertmanagerConfig_FromBundle_ValidSpec(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "AlertmanagerConfig", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	js := []byte(`{
		"apiVersion":"monitoring.coreos.com/v1alpha1",
		"kind":"AlertmanagerConfig",
		"spec":{
			"receivers":[{"name":"default"}],
			"route": {"receiver":"default"}
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("valid AlertmanagerConfig spec should pass: %v", err)
	}
}

func TestImport_AlertmanagerConfig_FromBundle_InvalidSpec_TypeMismatch(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "AlertmanagerConfig", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	// name should be string, give number
	bad := []byte(`{
		"apiVersion":"monitoring.coreos.com/v1alpha1",
		"kind":"AlertmanagerConfig",
		"spec":{
			"receivers":[{"name":1}]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected invalid_type for receivers[0].name")
	} else if iss, ok := goskema.AsIssues(err); ok {
		// Expect normalized JSON Pointer path after path normalization changes
		has := false
		for _, it := range iss {
			if it.Code != goskema.CodeInvalidType {
				continue
			}
			if it.Path == "/spec/receivers/0/name" {
				has = true
				break
			}
		}
		if !has {
			t.Fatalf("expected invalid_type at /spec/receivers/0/name; got: %v", iss)
		}
	}
}

func TestImport_Alertmanager_Email_Required_To(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_email_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
		"receivers": [
			{
				"emailConfigs": [
					{
						"to": "ops@example.com",
						"sendResolved": true
					}
				]
			}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected email to required satisfied: %v", err)
	}
	bad := []byte(`{
		"receivers": [
			{
				"emailConfigs": [
					{
						"sendResolved": true
					}
				]
			}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required to for emailConfigs[0]")
	}
}

func TestImport_Alertmanager_Webhook_Required_URL(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_webhook_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
		"receivers": [
			{
				"webhookConfigs": [
					{
						"url": "https://example.com/hook",
						"sendResolved": false
					}
				]
			}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected webhook url required satisfied: %v", err)
	}
	bad := []byte(`{
		"receivers": [
			{
				"webhookConfigs": [
					{
						"sendResolved": false
					}
				]
			}
		]
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required url for webhookConfigs[0]")
	}
}

func TestImport_Alertmanager_Slack_Required_Channel_And_ApiURLKey(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_slack_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
        "receivers":[{"slackConfigs":[{"channel":"#alert","apiURL":{"key":"slack-url"}}]}]
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected slack channel+apiURL.key required satisfied: %v", err)
	}
	bad1 := []byte(`{
        "receivers":[{"slackConfigs":[{"apiURL":{"key":"slack-url"}}]}]
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad1)); err == nil {
		t.Fatalf("expected required channel for slackConfigs[0]")
	}
	bad2 := []byte(`{
        "receivers":[{"slackConfigs":[{"channel":"#alert","apiURL":{}}]}]
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad2)); err == nil {
		t.Fatalf("expected required apiURL.key for slackConfigs[0]")
	}
}

func TestImport_Alertmanager_PagerDuty_Required_RoutingKey(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_pagerduty_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
        "receivers":[{"pagerdutyConfigs":[{"routingKey":"rk"}]}]
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected pagerduty routingKey required satisfied: %v", err)
	}
	bad := []byte(`{
        "receivers":[{"pagerdutyConfigs":[{}]}]
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required routingKey for pagerdutyConfigs[0]")
	}
}

func TestImport_Alertmanager_InhibitRules_NameRequired(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_inhibit_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{
		"spec": {
			"inhibitRules": [
				{
					"equal": ["job"],
					"sourceMatch": [
						{"name": "namespace", "value": "prod"}
					],
					"targetMatch": [
						{"name": "alertname", "value": "HighCPU"}
					]
				}
			]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected inhibitRules with name required satisfied: %v", err)
	}
	bad := []byte(`{
		"spec": {
			"inhibitRules": [
				{
					"sourceMatch": [
						{"value": "prod"}
					],
					"targetMatch": [
						{"name": "alertname"}
					]
				}
			]
		}
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required name in sourceMatch[0]")
	}
}

func TestImport_AlertmanagerConfig_Receivers_E2E_FromMiniBundle(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanagerconfig_receivers_email_webhook_slack_pagerduty.yaml")
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "AlertmanagerConfig", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	// minimal spec should pass
	base := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(base)); err != nil {
		t.Fatalf("base minimal spec should pass: %v", err)
	}
	// email ok
	emailOK := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"emailConfigs":[{"to":"ops@example.com"}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(emailOK)); err != nil {
		t.Fatalf("email ok should pass: %v", err)
	}
	// email bad (missing to)
	emailBad := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"emailConfigs":[{}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(emailBad)); err == nil {
		t.Fatalf("email missing to should fail")
	}
	// webhook ok
	whOK := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"webhookConfigs":[{"url":"https://example.com/h"}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(whOK)); err != nil {
		t.Fatalf("webhook ok should pass: %v", err)
	}
	// webhook bad
	whBad := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"webhookConfigs":[{}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(whBad)); err == nil {
		t.Fatalf("webhook missing url should fail")
	}
	// slack ok
	slackOK := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"slackConfigs":[{"channel":"#c","apiURL":{"key":"k"}}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(slackOK)); err != nil {
		t.Fatalf("slack ok should pass: %v", err)
	}
	// slack bad
	slackBad := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"slackConfigs":[{"channel":"#c","apiURL":{}}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(slackBad)); err == nil {
		t.Fatalf("slack missing apiURL.key should fail")
	}
	// pagerduty ok
	pdOK := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"pagerdutyConfigs":[{"routingKey":"rk"}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(pdOK)); err != nil {
		t.Fatalf("pagerduty ok should pass: %v", err)
	}
	// pagerduty bad
	pdBad := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{"receivers":[{"pagerdutyConfigs":[{}]}]}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(pdBad)); err == nil {
		t.Fatalf("pagerduty missing routingKey should fail")
	}
}

func TestImport_AlertmanagerConfig_InhibitRules_MatchTypePrecedenceAccept(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_inhibit_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// Both matchType and regex present; schema-level accepts both
	js := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "equal": [],
                    "sourceMatch": [
                        {"name": "namespace", "value": "prod", "regex": true, "matchType": "="}
                    ],
                    "targetMatch": [
                        {"name": "alertname", "value": "HighCPU", "regex": false, "matchType": "!="}
                    ]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("expected inhibitRules with matchType+regex accepted: %v", err)
	}
}

func TestImport_AlertmanagerConfig_InhibitRules_SourceTarget_RequiredAndType(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_inhibit_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}

	// Missing name in sourceMatch item -> required
	badMissingName := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "sourceMatch": [ {"value": "prod"} ],
                    "targetMatch": [ {"name": "alertname"} ]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badMissingName)); err == nil {
		t.Fatalf("expected required name in sourceMatch[0]")
	}

	// regex wrong type (string) -> invalid_type
	badRegexType := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "sourceMatch": [ {"name": "namespace", "value": "prod", "regex": "true"} ]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badRegexType)); err == nil {
		t.Fatalf("expected invalid_type for regex string")
	}

	// equal item wrong type (number) -> invalid_type
	badEqualType := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "equal": [1]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badEqualType)); err == nil {
		t.Fatalf("expected invalid_type for equal[0]")
	}

	// value wrong type (number) -> invalid_type
	badValueType := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "sourceMatch": [ {"name": "namespace", "value": 123} ]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badValueType)); err == nil {
		t.Fatalf("expected invalid_type for value number")
	}

	// Minimal equal: empty array accepted
	goodEmptyEqual := []byte(`{
        "spec": {
            "inhibitRules": [
                {
                    "equal": [],
                    "sourceMatch": [ {"name": "namespace"} ],
                    "targetMatch": [ {"name": "alertname"} ]
                }
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(goodEmptyEqual)); err != nil {
		t.Fatalf("expected empty equal accepted: %v", err)
	}
}
