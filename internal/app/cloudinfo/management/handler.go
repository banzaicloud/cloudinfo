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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"net/http"
)

type mngmntRouteHandler struct {
	cis cloudinfo.CloudInfoStore
}

func (mrh *mngmntRouteHandler) Export() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := mrh.cis.Export(); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, "export succeeded")

	}
}

func (mrh *mngmntRouteHandler) Import() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := mrh.cis.Import(); err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, "import succeeded")
	}
}

func (mrh *mngmntRouteHandler) Refresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get the provider
		//mrh.cis.Refresh()
	}
}

func SetupManagementEngine(cfg Config, cis cloudinfo.CloudInfoStore) *gin.Engine {
	if err := cfg.Validate(); err != nil {
		emperror.Panic(err)
	}

	rh := &mngmntRouteHandler{cis}

	router := gin.New()
	base := router.Group("/management/store")
	base.PUT("export", rh.Export())
	base.PUT("import", rh.Import())
	base.PUT("refresh/:provider", rh.Refresh())
	if err := router.Run(cfg.Address); err != nil {
		emperror.Panic(err)
	}

	return router
}
