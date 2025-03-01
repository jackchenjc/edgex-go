/*******************************************************************************
 * Copyright (C) 2024 IOTech Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/edgex-go/internal/support/scheduler/application"
	"github.com/edgexfoundry/edgex-go/internal/support/scheduler/container"
	"github.com/edgexfoundry/edgex-go/internal/support/scheduler/infrastructure"
)

// Bootstrap contains references to dependencies required by the BootstrapHandler.
type Bootstrap struct {
	router      *echo.Echo
	serviceName string
}

// NewBootstrap is a factory method that returns an initialized Bootstrap receiver struct.
func NewBootstrap(router *echo.Echo, serviceName string) *Bootstrap {
	return &Bootstrap{
		router:      router,
		serviceName: serviceName,
	}
}

// BootstrapHandler fulfills the BootstrapHandler contract and performs initialization needed by the scheduler service.
func (b *Bootstrap) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup, _ startup.Timer, dic *di.Container) bool {
	LoadRestRoutes(b.router, dic, b.serviceName)

	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	schedulerManager := infrastructure.NewManager(dic)
	dic.Update(di.ServiceConstructorMap{
		container.SchedulerManagerName: func(get di.Get) interface{} {
			return schedulerManager
		},
	})

	err := application.LoadScheduleJobsToSchedulerManager(ctx, dic)
	if err != nil {
		lc.Errorf("failed to load schedule jobs to scheduler manager: %v", err)
		return false
	}

	config := container.ConfigurationFrom(dic.Get)
	if config.Retention.Enabled {
		retentionInterval, err := time.ParseDuration(config.Retention.Interval)
		if err != nil {
			lc.Errorf("Failed to parse schedule action record retention interval, %v", err)
			return false
		}
		application.AsyncPurgeRecord(ctx, dic, retentionInterval)
	}

	return true
}
