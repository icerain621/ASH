package api

import (
	"net/http"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"

	"github.com/ash-repwiki/ash/internal/api/docs"
)

// registerSwagger mounts OpenAPI JSON and Swagger UI (appendix G).
func registerSwagger(r *gin.Engine) {
	r.GET("/openapi.json", func(c *gin.Context) {
		spec, err := swag.ReadDoc(docs.SwaggerInfo.InstanceName())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(spec))
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
		ginSwagger.DocExpansion("list"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}
