// Package docs provides OpenAPI (Swagger) metadata for the ASH Worker API.
//
// Regenerate with: make swagger
package docs

import (
	_ "embed"

	"github.com/swaggo/swag"
)

//go:embed swagger.json
var docTemplate string

// SwaggerInfo holds exported Swagger metadata.
var SwaggerInfo = &swag.Spec{
	Version:          "0.1",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "ASH Worker API",
	Description:      "Delivery-oriented coding robot worker API (M0). See docs/appendices/G-OpenAPI-端点清单(M0).md.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
