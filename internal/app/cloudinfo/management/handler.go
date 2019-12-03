// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package management

import (
	"context"
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/api"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

// mngmntRouteHandler struct collecting handlers for the management service
type mngmntRouteHandler struct {
	cis cloudinfo.CloudInfoStore
	sd  cloudinfo.ScrapingDriver
	log cloudinfo.Logger
}

// Export exports the content of the Store into the response body
func (mrh *mngmntRouteHandler) Export() gin.HandlerFunc {
	return func(c *gin.Context) {
		mrh.log.Info("exporting cloud information")
		if err := mrh.cis.Export(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.Writer.WriteHeader(http.StatusOK)
	}
}

// Import loads the data into the underlying store implementation from the stdin
func (mrh *mngmntRouteHandler) Import() gin.HandlerFunc {

	return func(c *gin.Context) {
		mrh.log.Info("importing cloud information")
		f, fh, err := c.Request.FormFile("data")
		if err != nil {
			mrh.log.Error("failed to import data", map[string]interface{}{"err": err})
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		mrh.log.Info("loading cloud info", map[string]interface{}{"file": fh.Filename, "size": fh.Size})
		if err := mrh.cis.Import(f); err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, "import succeeded")
	}
}

// Refresh handler that triggers the refresh process for a provider
func (mrh *mngmntRouteHandler) Refresh() gin.HandlerFunc {

	return func(c *gin.Context) {
		pathParams := api.GetProviderPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			mrh.log.Error("failed to get provider")
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get the provider from path"})
			return
		}
		// trigger refreshing the provider
		if pathParams.Provider == "" {
			mrh.log.Error("failed to get provider")
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get the provider from path"})
			return
		}

		// trigger the refresh process for the provider
		mrh.log.Info("triggering refresh cloud information", map[string]interface{}{"provider": pathParams.Provider})
		go mrh.sd.RefreshProvider(context.Background(), pathParams.Provider)
		c.JSON(http.StatusOK, gin.H{"operation": "refresh", "provider": pathParams.Provider})
	}
}

func StartManagementEngine(cfg Config, cis cloudinfo.CloudInfoStore, sd cloudinfo.ScrapingDriver, log cloudinfo.Logger) *gin.Engine {
	if err := cfg.Validate(); err != nil {
		emperror.Panic(err)
	}

	rh := &mngmntRouteHandler{cis, sd, log}

	router := gin.New()
	base := router.Group("/management/store")
	base.GET("export", rh.Export())
	base.PUT("import", rh.Import())
	base.PUT("refresh/:provider", rh.Refresh())
	if err := router.Run(cfg.Address); err != nil {
		emperror.Panic(err)
	}

	return router
}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
