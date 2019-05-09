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

package cloudinfodriver

import (
	"context"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

const (
	OperationServiceListServices = "cloudinfo.Service.ListServices"
)

// ServiceService returns the list of supported services.
type ServiceService interface {
	// ListServices returns a list of services supported by a provider.
	ListServices(ctx context.Context, provider string) ([]cloudinfo.Service, error)
}
