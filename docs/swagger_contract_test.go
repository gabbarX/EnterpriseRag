package docs_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	docs "github.com/ORG_PLACEHOLDER/EnterpriseRag/docs"
	apperrors "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/errors"
	"gopkg.in/yaml.v3"
)

type swaggerDocumentCase struct {
	name     string
	loadSpec func(t *testing.T) []byte
	parse    func([]byte, any) error
}

type swaggerParameter struct {
	Name     string `json:"name" yaml:"name"`
	In       string `json:"in" yaml:"in"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
}

type swaggerOperation struct {
	Parameters []swaggerParameter `json:"parameters" yaml:"parameters"`
}

func swaggerDocuments() []swaggerDocumentCase {
	return []swaggerDocumentCase{
		{
			name: "registered document",
			loadSpec: func(t *testing.T) []byte {
				t.Helper()
				return []byte(docs.SwaggerInfo.ReadDoc())
			},
			parse: json.Unmarshal,
		},
		{
			name: "swagger.json",
			loadSpec: func(t *testing.T) []byte {
				t.Helper()
				return readSwaggerFile(t, "swagger.json")
			},
			parse: json.Unmarshal,
		},
		{
			name: "swagger.yaml",
			loadSpec: func(t *testing.T) []byte {
				t.Helper()
				return readSwaggerFile(t, "swagger.yaml")
			},
			parse: yaml.Unmarshal,
		},
	}
}

func TestKnowledgeSearchRouteContract(t *testing.T) {
	for _, tt := range swaggerDocuments() {
		t.Run(tt.name, func(t *testing.T) {
			assertKnowledgeSearchRouteContract(t, tt.loadSpec(t), tt.parse)
		})
	}
}

func TestModelDeleteUsageContract(t *testing.T) {
	for _, tt := range swaggerDocuments() {
		t.Run(tt.name, func(t *testing.T) {
			assertModelDeleteUsageContract(t, tt.loadSpec(t), tt.parse)
		})
	}
}

func TestFAQEnabledFilterContract(t *testing.T) {
	for _, tt := range swaggerDocuments() {
		t.Run(tt.name, func(t *testing.T) {
			assertFAQEnabledFilterContract(t, tt.loadSpec(t), tt.parse)
		})
	}
}

func readSwaggerFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}

func assertKnowledgeSearchRouteContract(t *testing.T, data []byte, parse func([]byte, any) error) {
	t.Helper()
	var spec struct {
		Paths map[string]map[string]any `json:"paths" yaml:"paths"`
	}
	if err := parse(data, &spec); err != nil {
		t.Fatalf("parse generated Swagger document: %v", err)
	}

	knowledgeSearch, ok := spec.Paths["/knowledge-search"]
	if !ok {
		t.Fatal("generated Swagger document does not expose /knowledge-search")
	}
	if _, ok := knowledgeSearch["post"]; !ok {
		t.Fatal("generated Swagger document does not expose POST /knowledge-search")
	}

	if staleRoute, ok := spec.Paths["/sessions/search"]; ok {
		if _, ok := staleRoute["post"]; ok {
			t.Fatal("generated Swagger document still exposes stale POST /sessions/search")
		}
	}
}

func assertModelDeleteUsageContract(t *testing.T, data []byte, parse func([]byte, any) error) {
	t.Helper()
	var spec struct {
		Paths map[string]map[string]struct {
			Responses map[string]any `json:"responses" yaml:"responses"`
		} `json:"paths" yaml:"paths"`
		Definitions map[string]map[string]any `json:"definitions" yaml:"definitions"`
	}
	if err := parse(data, &spec); err != nil {
		t.Fatalf("parse generated Swagger document: %v", err)
	}

	models, ok := spec.Paths["/models/{id}"]
	if !ok {
		t.Fatal("generated Swagger document does not expose /models/{id}")
	}
	deleteOperation, ok := models["delete"]
	if !ok {
		t.Fatal("generated Swagger document does not expose DELETE /models/{id}")
	}
	if _, ok := deleteOperation.Responses["400"]; !ok {
		t.Fatal("model DELETE Swagger contract does not document the model-in-use 400 response")
	}

	// The 400 body must be documented as an AppError, which is what carries the
	// numeric error code the client switches on.
	if !swaggerResponseRefersToAppError(deleteOperation.Responses["400"]) {
		t.Fatalf("model DELETE 400 response does not reference an AppError schema: %#v",
			deleteOperation.Responses["400"])
	}
	for _, name := range []string{"errors.AppError", "github_com_ORG_PLACEHOLDER_EnterpriseRag_internal_errors.AppError"} {
		if _, ok := spec.Definitions[name]; ok {
			return
		}
	}
	t.Fatalf("generated Swagger document does not define an AppError schema")
}

// swaggerResponseRefersToAppError reports whether a generated response body is
// documented with the AppError schema, under either the short or the fully
// qualified definition name swag may choose.
func swaggerResponseRefersToAppError(response any) bool {
	body, ok := response.(map[string]any)
	if !ok {
		return false
	}
	schema, ok := body["schema"].(map[string]any)
	if !ok {
		return false
	}
	ref, ok := schema["$ref"].(string)
	return ok && strings.HasSuffix(ref, "errors.AppError")
}

// TestModelInUseErrorCodeContract pins the numeric code the model DELETE 400
// carries. It is asserted against the Go constant rather than the generated
// document because swag no longer emits a standalone ErrorCode enum for the
// AppError.Code field - it renders it as a bare integer.
func TestModelInUseErrorCodeContract(t *testing.T) {
	if got := int(apperrors.ErrModelInUse); got != 2300 {
		t.Fatalf("ErrModelInUse must stay 2300 (the value clients switch on), got %d", got)
	}
}

func assertFAQEnabledFilterContract(t *testing.T, data []byte, parse func([]byte, any) error) {
	t.Helper()
	var spec struct {
		Paths map[string]map[string]swaggerOperation `json:"paths" yaml:"paths"`
	}
	if err := parse(data, &spec); err != nil {
		t.Fatalf("parse generated Swagger document: %v", err)
	}

	faqEntries, ok := spec.Paths["/knowledge-bases/{id}/faq/entries"]
	if !ok {
		t.Fatal("generated Swagger document does not expose FAQ entries route")
	}
	getOperation, ok := faqEntries["get"]
	if !ok {
		t.Fatal("generated Swagger document does not expose GET FAQ entries")
	}
	for _, parameter := range getOperation.Parameters {
		if parameter.Name != "is_enabled" {
			continue
		}
		if parameter.In != "query" || parameter.Type != "boolean" || parameter.Required {
			t.Fatalf("is_enabled must be an optional boolean query parameter, got %#v", parameter)
		}
		return
	}
	t.Fatal("GET FAQ entries does not document the is_enabled query parameter")
}
