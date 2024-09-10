//
// Copyright (C) 2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"
	"time"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"

	"github.com/edgexfoundry/edgex-go/internal/pkg/correlation"
	"github.com/edgexfoundry/edgex-go/internal/support/cronscheduler/container"
	"github.com/edgexfoundry/edgex-go/internal/support/cronscheduler/infrastructure/interfaces"
)

// AddScheduleJob adds a new schedule job
func AddScheduleJob(ctx context.Context, job models.ScheduleJob, dic *di.Container) (string, errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	correlationId := correlation.FromContext(ctx)

	// Add the ID for each action
	for i, action := range job.Actions {
		job.Actions[i] = action.WithId("")
	}

	err := schedulerManager.AddScheduleJob(job, correlationId)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	addedJob, err := dbClient.AddScheduleJob(ctx, job)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	arrangeScheduleJob(ctx, job, dic)

	lc.Debugf("Successfully created the scheduled job. ScheduleJob ID: %s, Correlation-ID: %s", addedJob.Id, correlationId)
	return addedJob.Id, nil
}

// TriggerScheduleJobByName triggers a schedule job by name
func TriggerScheduleJobByName(ctx context.Context, name string, dic *di.Container) errors.EdgeX {
	if name == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}

	correlationId := correlation.FromContext(ctx)
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	err := schedulerManager.TriggerScheduleJobByName(name, correlationId)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Successfully triggered the scheduled job. Correlation-ID: %s", correlationId)
	return nil
}

// ScheduleJobByName queries the schedule job by name
func ScheduleJobByName(ctx context.Context, name string, dic *di.Container) (dto dtos.ScheduleJob, edgeXerr errors.EdgeX) {
	if name == "" {
		return dto, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)
	job, err := dbClient.ScheduleJobByName(ctx, name)
	if err != nil {
		return dto, errors.NewCommonEdgeXWrapper(err)
	}
	dto = dtos.FromScheduleJobModelToDTO(job)

	return dto, nil
}

// AllScheduleJobs queries all the schedule jobs with offset and limit
func AllScheduleJobs(ctx context.Context, labels []string, offset, limit int, dic *di.Container) (scheduleJobDTOs []dtos.ScheduleJob, totalCount uint32, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	jobs, err := dbClient.AllScheduleJobs(ctx, labels, offset, limit)
	if err == nil {
		totalCount, err = dbClient.ScheduleJobTotalCount(ctx, labels)
	}
	if err != nil {
		return scheduleJobDTOs, totalCount, errors.NewCommonEdgeXWrapper(err)
	}

	scheduleJobDTOs = make([]dtos.ScheduleJob, len(jobs))
	for i, job := range jobs {
		dto := dtos.FromScheduleJobModelToDTO(job)
		scheduleJobDTOs[i] = dto
	}

	return scheduleJobDTOs, totalCount, nil
}

// PatchScheduleJob executes the PATCH operation with the DTO to replace the old data
func PatchScheduleJob(ctx context.Context, dto dtos.UpdateScheduleJob, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	correlationId := correlation.FromContext(ctx)

	job, err := scheduleJobByDTO(ctx, dbClient, dto)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	requests.ReplaceScheduleJobModelFieldsWithDTO(&job, dto)

	// Add the ID for each action, the old actions will be replaced by the new actions
	for i, action := range job.Actions {
		job.Actions[i] = action.WithId("")
	}

	err = schedulerManager.UpdateScheduleJob(job, correlationId)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	err = dbClient.UpdateScheduleJob(ctx, job)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	arrangeScheduleJob(ctx, job, dic)

	lc.Debugf("Successfully patched the scheduled job: %s. ScheduleJob ID: %s, Correlation-ID: %s", job.Name, job.Id, correlationId)
	return nil
}

func scheduleJobByDTO(ctx context.Context, dbClient interfaces.DBClient, dto dtos.UpdateScheduleJob) (job models.ScheduleJob, err errors.EdgeX) {
	// The ID or Name is required by DTO and the DTO also accepts empty string ID if the Name is provided
	if dto.Id != nil && *dto.Id != "" {
		job, err = dbClient.ScheduleJobById(ctx, *dto.Id)
		if err != nil {
			return job, errors.NewCommonEdgeXWrapper(err)
		}
	} else {
		job, err = dbClient.ScheduleJobByName(ctx, *dto.Name)
		if err != nil {
			return job, errors.NewCommonEdgeXWrapper(err)
		}
	}
	if dto.Name != nil && *dto.Name != job.Name {
		return job, errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("scheduled job name '%s' not match the exsting '%s' ", *dto.Name, job.Name), nil)
	}
	return job, nil
}

// DeleteScheduleJobByName deletes the schedule job by name
func DeleteScheduleJobByName(ctx context.Context, name string, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	correlationId := correlation.FromContext(ctx)

	err := schedulerManager.DeleteScheduleJobByName(name, correlationId)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	err = dbClient.DeleteScheduleJobByName(ctx, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Successfully deleted the scheduled job: %s. Correlation-ID: %s", name, correlationId)
	return nil
}

// LoadScheduleJobsToSchedulerManager loads all the existing schedule jobs to the scheduler manager, the MaxResultCount config is used to limit the number of jobs that will be loaded
func LoadScheduleJobsToSchedulerManager(ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	ctx, correlationId := correlation.FromContextOrNew(ctx)
	config := container.ConfigurationFrom(dic.Get)

	jobs, err := dbClient.AllScheduleJobs(context.Background(), nil, 0, config.Service.MaxResultCount)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to load all existing scheduled jobs", err)
	}

	for _, job := range jobs {
		err := schedulerManager.AddScheduleJob(job, correlationId)
		if err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}

		// Load the existing scheduled jobs to the scheduler manager
		arrangeScheduleJob(ctx, job, dic)

		// Generate missed schedule action records for the existing scheduled jobs
		err = generateMissedRecords(ctx, job, dic)
		if err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}

		lc.Debugf("Successfully loaded the existing scheduled job: %s. Correlation-ID: %s", job.Name, correlationId)
	}

	return nil
}

// arrangeScheduleJob arranges the schedule job based on the startTimestamp and endTimestamp
func arrangeScheduleJob(ctx context.Context, job models.ScheduleJob, dic *di.Container) {
	schedulerManager := container.SchedulerManagerFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	correlationId := correlation.FromContext(ctx)

	if job.AdminState != models.Unlocked {
		lc.Debugf("The scheduled job is ready but not started because the admin state is locked. ScheduleJob ID: %s, Correlation-ID: %s", job.Id, correlationId)
		return
	}

	startTimestamp := job.Definition.GetBaseScheduleDef().StartTimestamp
	endTimestamp := job.Definition.GetBaseScheduleDef().EndTimestamp

	durationUntilStart := time.Until(time.UnixMilli(startTimestamp))
	durationUntilEnd := time.Until(time.UnixMilli(endTimestamp))
	isEndTimestampExpired := endTimestamp != 0 && durationUntilEnd < 0

	// If endTimestamp is set and expired, the scheduled job should not be triggered
	if isEndTimestampExpired {
		lc.Warnf("The endTimestamp is expired for the scheduled job: %s, which will not be started. Correlation-ID: %s", job.Name, correlationId)
		return
	}

	// If startTimestamp is expired, the scheduled job should be started immediately
	if durationUntilStart < 0 {
		lc.Debugf("The startTimestamp is expired for the scheduled job: %s, which will be started immediately. Correlation-ID: %s", job.Name, correlationId)
		durationUntilStart = 0
	} else if durationUntilStart > 0 {
		lc.Debugf("The scheduled job: %s will be started at %v (timestamp: %v). Correlation-ID: %s", job.Name, time.UnixMilli(startTimestamp), startTimestamp, correlationId)
	}

	// Regardless of whether startTimestamp has a value or not, the job should always be started by default if endTimestamp is not expired.
	time.AfterFunc(durationUntilStart, func() {
		err := schedulerManager.StartScheduleJobByName(job.Name, correlationId)
		if err != nil {
			lc.Errorf("Failed to start the scheduled job: %s based on startTimestamp. Error: %v, Correlation-ID: %s", job.Name, err, correlationId)
		}
	})

	// If the endTimestamp is set and the duration until the end is greater than 0, the scheduled job will be stopped at the endTimestamp
	if endTimestamp != 0 && durationUntilEnd > 0 {
		lc.Debugf("The scheduled job: %s will be stopped at %v (timestamp: %v). Correlation-ID: %s", job.Name, time.UnixMilli(endTimestamp), endTimestamp, correlationId)
		time.AfterFunc(durationUntilEnd, func() {
			err := schedulerManager.StopScheduleJobByName(job.Name, correlationId)
			if err != nil {
				lc.Errorf("Failed to stop the scheduled job: %s based on endTimestamp. Error: %v, Correlation-ID: %s", job.Name, err, correlationId)
			}
		})
	}
}

// generateMissedRecords generates missed schedule action records
func generateMissedRecords(ctx context.Context, job models.ScheduleJob, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	correlationId := correlation.FromContext(ctx)

	if job.AdminState != models.Unlocked {
		lc.Debugf("The scheduled job: %s is locked, skip generating missed schedule action records. ScheduleJob ID: %s, Correlation-ID: %s", job.Name, job.Id, correlationId)
		return nil
	}

	// Get the latest schedule action records by job name and generate missed schedule action records
	latestRecords, err := dbClient.LatestScheduleActionRecordsByJobName(ctx, job.Name)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, fmt.Sprintf("failed to load the latest schedule action records of job: %s", job.Name), err)
	}
	err = GenerateMissedScheduleActionRecords(ctx, dic, job, latestRecords)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	return nil
}