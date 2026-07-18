package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositorySourceAndBundlePassStrictLint(t *testing.T) {
	for _, root := range []string{
		filepath.Join("..", "..", "openapi", "openapi.yaml"),
		filepath.Join("..", "..", "openapi", "generated", "openapi.bundle.yaml"),
	} {
		t.Run(filepath.Base(root), func(t *testing.T) {
			diagnostics, err := Lint(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diagnostics[:min(10, len(diagnostics))])
			}
		})
	}
}

func TestStrictLintMutationFamilies(t *testing.T) {
	for _, test := range strictLintMutationCases {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, test.old, test.replacement, 1)
			if source == strictLintFixture {
				t.Fatalf("fixture does not contain %q", test.old)
			}
			root := filepath.Join(t.TempDir(), "openapi.yaml")
			if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			diagnostics, err := Lint(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Rule == test.rule {
					if diagnostic.Artifact == "" || diagnostic.Location == "" {
						t.Fatalf("incomplete diagnostic: %+v", diagnostic)
					}
					return
				}
			}
			t.Fatalf("missing %s diagnostic: %+v", test.rule, diagnostics)
		})
	}
}

type strictLintMutation struct {
	name        string
	old         string
	replacement string
	rule        string
	goOnly      bool
}

var strictLintMutationCases = []strictLintMutation{
	{name: "operation id required", old: "      operationId: getThing\n", rule: "operation-operationId"},
	{name: "operation id safe", old: "operationId: getThing", replacement: "operationId: get thing", rule: "operation-operationId-url-safe"},
	{name: "summary required", old: "      summary: Get a thing\n", rule: "operation-summary"},
	{name: "success response", old: "        '200':", replacement: "        '400':", rule: "operation-2xx-response"},
	{name: "response description", old: "          description: OK\n", rule: "response-description"},
	{name: "default response content", old: "        '200':\n          description: OK\n          content:\n            application/json:", replacement: "        default:\n          description: OK\n          content:\n            not-a-media-type:", rule: "media-type-name", goOnly: true},
	{name: "path parameter", old: "        - name: id", replacement: "        - name: other", rule: "path-parameters-defined"},
	{name: "duplicate parameter", old: "        - name: id\n          in: path\n          required: true\n          schema:\n            type: string", replacement: "        - name: id\n          in: path\n          required: true\n          schema:\n            type: string\n        - name: id\n          in: path\n          required: true\n          schema:\n            type: string", rule: "parameter-unique"},
	{name: "unknown tag", old: "tags: [things]", replacement: "tags: [missing]", rule: "operation-tag-defined", goOnly: true},
	{name: "tag description", old: "    description: Things\n", rule: "tag-description"},
	{name: "security scheme", old: "  - apiKey: []", replacement: "  - missing: []", rule: "security-defined"},
	{name: "server trailing slash", old: "https://api.invalid", replacement: "https://api.invalid/", rule: "no-server-trailing-slash"},
	{name: "server example", old: "https://api.invalid", replacement: "https://example.com", rule: "no-server-example.com"},
	{name: "server example with userinfo", old: "https://api.invalid", replacement: "https://user@example.com", rule: "no-server-example.com", goOnly: true},
	{name: "required property", old: "required: [id]", replacement: "required: [missing]", rule: "no-required-schema-properties-undefined"},
	{name: "required without properties", old: "    Thing:\n      type: object", replacement: "    MissingProperties:\n      type: object\n      required: [missing]\n    Thing:\n      type: object", rule: "no-required-schema-properties-undefined"},
	{name: "enum type", old: "enum: [one]", replacement: "enum: [1]", rule: "no-enum-type-mismatch"},
	{name: "schema range", old: "maxLength: 10", replacement: "maxLength: 0", rule: "no-schema-type-mismatch"},
	{name: "schema example", old: "example: one", replacement: "example: 1", rule: "no-invalid-schema-examples"},
	{name: "parameter example", old: "            type: string\n      responses:", replacement: "            type: string\n          example: 1\n      responses:", rule: "no-invalid-parameter-examples"},
	{name: "media example", old: "                $ref: '#/components/schemas/Thing'", replacement: "                $ref: '#/components/schemas/Thing'\n              example: {id: 1}", rule: "no-invalid-media-type-examples"},
	{name: "media encoding", old: "                $ref: '#/components/schemas/Thing'", replacement: "                $ref: '#/components/schemas/Thing'\n              encoding:\n                missing: {}", rule: "no-invalid-media-type-encoding", goOnly: true},
	{name: "unused component", old: "    Thing:", replacement: "    Unused: {type: string}\n    Thing:", rule: "no-unused-components"},
	{name: "empty servers", old: "servers:\n  - url: https://api.invalid\n    description: Production", replacement: "servers: []", rule: "no-empty-servers"},
	{name: "server variable default", old: "  - url: https://api.invalid\n    description: Production", replacement: "  - url: https://{region}.api.invalid\n    description: Production\n    variables:\n      region:\n        default: kr\n        enum: [us]", rule: "server-variables-empty-enum"},
	{name: "tag parent cycle", old: "  - name: things\n    description: Things", replacement: "  - name: things\n    description: Things\n    parent: things", rule: "no-invalid-tag-parents"},
	{name: "mixed number range", old: "          minLength: 1", replacement: "          minimum: 1\n          exclusiveMinimum: 2\n          minLength: 1", rule: "no-mixed-number-range-constraints"},
	{name: "discriminator default", old: "      properties:\n        id:", replacement: "      discriminator: {propertyName: kind}\n      properties:\n        kind: {type: string}\n        id:", rule: "discriminator-defaultMapping"},
	{name: "discriminator missing mapping", old: "      properties:\n        id:", replacement: "      discriminator: {propertyName: id, defaultMapping: '#/components/schemas/Missing'}\n      properties:\n        id:", rule: "discriminator-defaultMapping"},
	{name: "discriminator unresolved file", old: "      properties:\n        id:", replacement: "      discriminator: {propertyName: id, defaultMapping: './missing.yaml#/Nope'}\n      properties:\n        id:", rule: "discriminator-defaultMapping"},
	{name: "nested schema", old: "      properties:\n        id:", replacement: "      $defs:\n        Bad:\n          type: object\n          required: [missing]\n          properties: {}\n      properties:\n        id:", rule: "no-required-schema-properties-undefined"},
	{name: "schema example constraint", old: "example: one", replacement: "example: far-too-long-example", rule: "no-invalid-schema-examples"},
	{name: "media example enum", old: "                $ref: '#/components/schemas/Thing'", replacement: "                $ref: '#/components/schemas/Thing'\n              example: {id: two}", rule: "no-invalid-media-type-examples"},
	{name: "webhook operation", old: "components:\n", replacement: "webhooks:\n  changed:\n    post:\n      operationId: changedWebhook\n      responses:\n        '200': {description: OK}\ncomponents:\n", rule: "operation-summary"},
	{name: "callback operation", old: "      responses:\n", replacement: "      callbacks:\n        changed:\n          '{$request.body#/callbackUrl}':\n            post:\n              operationId: changedCallback\n              responses:\n                '200': {description: OK}\n      responses:\n", rule: "operation-summary"},
	{name: "component path item operation", old: "  schemas:\n", replacement: "  pathItems:\n    Changed:\n      post:\n        operationId: changedPathItem\n        responses:\n          '200': {description: OK}\n  schemas:\n", rule: "operation-summary"},
}

var strictValidationMutationCases = []strictLintMutation{
	{name: "server variable enum", old: "  - url: https://api.invalid\n    description: Production", replacement: "  - url: https://{region}.api.invalid\n    description: Production\n    variables:\n      region:\n        default: kr\n        enum: []"},
	{name: "querystring parameters", old: "      responses:", replacement: "        - name: raw\n          in: querystring\n          schema: {type: string}\n        - name: filter\n          in: query\n          schema: {type: string}\n      responses:"},
	{name: "encoding combinations", old: "                $ref: '#/components/schemas/Thing'", replacement: "                $ref: '#/components/schemas/Thing'\n              encoding: {id: {}}\n              itemEncoding: {id: {}}"},
	{name: "prefix encoding combinations", old: "                $ref: '#/components/schemas/Thing'", replacement: "                $ref: '#/components/schemas/Thing'\n              encoding: {id: {}}\n              prefixEncoding: [{id: {}}]"},
	{name: "example value forms", old: "          schema:\n            type: string\n      responses:", replacement: "          schema:\n            type: string\n          examples:\n            conflict:\n              value: one\n              externalValue: https://api.invalid/example\n      responses:"},
}

func TestStrictLintStructuralValidationFamilies(t *testing.T) {
	for _, mutation := range strictValidationMutationCases {
		t.Run(mutation.name, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, mutation.old, mutation.replacement, 1)
			root := filepath.Join(t.TempDir(), "openapi.yaml")
			if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Lint(root); err == nil {
				t.Fatal("structurally invalid mutation passed")
			}
		})
	}
}

func TestStrictLintServerHostSemantics(t *testing.T) {
	source := strings.Replace(strictLintFixture, "https://api.invalid", "https://notexample.com", 1)
	diagnostics := lintFixtureSource(t, source)
	for _, diagnostic := range diagnostics {
		if diagnostic.Rule == "no-server-example.com" {
			t.Fatalf("near-match host was rejected: %+v", diagnostic)
		}
	}

	for _, serverURL := range []string{
		"http://localhost:8080",
		"https://user@example.com",
		"https://api.example.com.",
	} {
		t.Run(serverURL, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, "https://api.invalid", serverURL, 1)
			assertLintRule(t, lintFixtureSource(t, source), "no-server-example.com")
		})
	}
}

func TestStrictLintDefaultResponseContent(t *testing.T) {
	source := strings.Replace(strictLintFixture, "        '200':", "        default:", 1)
	source = strings.Replace(source, "            application/json:", "            not-a-media-type:", 1)
	diagnostics := lintFixtureSource(t, source)
	assertLintRuleAt(t, diagnostics, "media-type-name", "/responses/default/content/")
}

func TestStrictLintRequiredWithoutProperties(t *testing.T) {
	source := strings.Replace(strictLintFixture,
		"    Thing:\n      type: object",
		"    MissingProperties:\n      type: object\n      required: [missing]\n    Thing:\n      type: object",
		1,
	)
	diagnostics := lintFixtureSource(t, source)
	assertLintRuleAt(t, diagnostics, "no-required-schema-properties-undefined", "/components/schemas/MissingProperties/required")
}

func TestStrictLintTraversesNonPathOperations(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		insert   string
		location string
	}{
		{
			name: "webhook",
			old:  "components:\n",
			insert: "webhooks:\n  changed:\n    post:\n      operationId: changedWebhook\n" +
				"      responses:\n        '200': {description: OK}\ncomponents:\n",
			location: "/webhooks/changed/post",
		},
		{
			name: "operation callback",
			old:  "      responses:\n",
			insert: "      callbacks:\n        changed:\n          '{$request.body#/callbackUrl}':\n            post:\n" +
				"              operationId: changedCallback\n              responses:\n                '200': {description: OK}\n      responses:\n",
			location: "/callbacks/changed/",
		},
		{
			name: "component path item",
			old:  "  schemas:\n",
			insert: "  pathItems:\n    Changed:\n      post:\n        operationId: changedPathItem\n" +
				"        responses:\n          '200': {description: OK}\n  schemas:\n",
			location: "/components/pathItems/Changed/post",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, test.old, test.insert, 1)
			if source == strictLintFixture {
				t.Fatalf("fixture does not contain %q", test.old)
			}
			diagnostics := lintFixtureSource(t, source)
			assertLintRuleAt(t, diagnostics, "operation-summary", test.location)
		})
	}
}

func TestStrictLintAllowsEncodingNamesOutsideMediaTypes(t *testing.T) {
	source := strictLintFixture + "x-example:\n  content:\n    application/json:\n      encoding: {}\n      itemEncoding: {}\n"
	diagnostics := lintFixtureSource(t, source)
	assertNoLintRule(t, diagnostics, "invalid-encoding-combinations")
}

func TestStrictLintComposedRequiredDiscriminator(t *testing.T) {
	source := strings.Replace(strictLintFixture, "      properties:\n        id:", "      discriminator: {propertyName: kind}\n      allOf:\n        - type: object\n          required: [kind]\n      properties:\n        kind: {type: string}\n        id:", 1)
	diagnostics := lintFixtureSource(t, source)
	assertNoLintRule(t, diagnostics, "discriminator-defaultMapping")
}

func TestStrictLintCountsWebhookSecurityUsage(t *testing.T) {
	source := strings.Replace(strictLintFixture, "components:\n  securitySchemes:\n", "webhooks:\n  callback:\n    post:\n      operationId: callback\n      summary: Callback\n      security: [{callbackKey: []}]\n      responses:\n        '200': {description: OK}\ncomponents:\n  securitySchemes:\n    callbackKey:\n      type: apiKey\n      in: header\n      name: X-Callback-Key\n", 1)
	diagnostics := lintFixtureSource(t, source)
	for _, diagnostic := range diagnostics {
		if diagnostic.Rule == "no-unused-components" && strings.HasSuffix(diagnostic.Location, "/callbackKey") {
			t.Fatalf("webhook security usage was ignored: %+v", diagnostic)
		}
	}
}

func lintFixtureSource(t *testing.T, source string) []LintDiagnostic {
	t.Helper()
	root := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := Lint(root)
	if err != nil {
		t.Fatal(err)
	}
	return diagnostics
}

func assertNoLintRule(t *testing.T, diagnostics []LintDiagnostic, rule string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Rule == rule {
			t.Fatalf("unexpected %s diagnostic: %+v", rule, diagnostic)
		}
	}
}

func assertLintRule(t *testing.T, diagnostics []LintDiagnostic, rule string) {
	t.Helper()
	assertLintRuleAt(t, diagnostics, rule, "")
}

func assertLintRuleAt(t *testing.T, diagnostics []LintDiagnostic, rule, location string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Rule == rule && strings.Contains(diagnostic.Location, location) {
			return
		}
	}
	t.Fatalf("missing %s diagnostic at %q: %+v", rule, location, diagnostics)
}

func TestStrictLintRejectsOpenAPISchemaMutation(t *testing.T) {
	source := strings.Replace(strictLintFixture, "  schemas:\n    Thing:", "  schemas:\n    'bad name': {type: string}\n    Thing:", 1)
	root := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Lint(root); err == nil {
		t.Fatal("invalid component name passed OpenAPI schema validation")
	}
}

const strictLintFixture = `openapi: 3.2.0
info:
  title: Strict lint fixture
  version: 1.0.0
servers:
  - url: https://api.invalid
    description: Production
tags:
  - name: things
    description: Things
security:
  - apiKey: []
paths:
  /things/{id}:
    get:
      operationId: getThing
      summary: Get a thing
      tags: [things]
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Thing'
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: query
      name: key
  schemas:
    Thing:
      type: object
      required: [id]
      properties:
        id:
          type: string
          minLength: 1
          maxLength: 10
          enum: [one]
          example: one
`
