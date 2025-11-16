package api

import (
	"embed"
	"fmt"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	//go:embed openapi.yaml
	specFS embed.FS

	swaggerOnce sync.Once
	swaggerSpec *openapi3.T
	swaggerErr  error
)

// GetSwagger returns compiled OpenAPI specification used by runtime validators.
func GetSwagger() (*openapi3.T, error) {
	swaggerOnce.Do(func() {
		specBytes, err := specFS.ReadFile("openapi.yaml")
		if err != nil {
			swaggerErr = fmt.Errorf("read embedded openapi spec: %w", err)
			return
		}

		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true

		swaggerSpec, swaggerErr = loader.LoadFromData(specBytes)
	})
	if swaggerErr != nil {
		return nil, swaggerErr
	}
	return swaggerSpec, nil
}
