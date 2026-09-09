// Package docs embeds the service's OpenAPI contract in the binary.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte
