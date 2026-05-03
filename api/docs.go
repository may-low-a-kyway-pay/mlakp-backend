package api

import _ "embed"

// OpenAPIYAML is the generated API contract served by /docs/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
