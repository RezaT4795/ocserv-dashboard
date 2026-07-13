package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	commonModels "github.com/mmtaee/ocserv-dashboard/common/models"
	occtlDocker "github.com/mmtaee/ocserv-dashboard/common/occtl_docker"
	"github.com/mmtaee/ocserv-dashboard/common/ocserv/occtl"
	"github.com/mmtaee/ocserv-dashboard/common/ocserv/user"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/database"
	"github.com/mmtaee/ocserv-dashboard/common/pkg/logger"
	"github.com/mmtaee/ocserv-dashboard/user_expiry/internal/models"
	stateManager "github.com/mmtaee/ocserv-dashboard/user_expiry/pkg/state"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type userAction func(string) (string, error)

// CornService handles all scheduled background jobs related to
// user expiration, monthly reactivation and auto-deletion.
//
// It supports both docker-mode and native ocserv mode.

type CornService struct {
	disconnectUser userAction
	lockUser       userAction
	unlockUser     userAction
}

// NewCornService initializes cron service.
// If dockerMode is true, docker-based occtl commands will be used.
// Otherwise, native ocserv handlers are used.
func NewCornService(dockerMode bool) *CornService {
	if dockerMode {
		dockerRepo := occtlDocker.NewOcservOcctlDocker()

		return &CornService{
			disconnectUser: dockerRepo.DisconnectUser,
			lockUser:       dockerRepo.Lock,
			unlockUser:     dockerRepo.Unlock,
		}
	}

	occtlHandler := occtl.NewOcservOcctl()
	ocservUserHandler := user.NewOcservUser()

	return &CornService{
		disconnectUser: occtlHandler.DisconnectUser,
		lockUser:       ocservUserHandler.Lock,
		unlockUser:     ocservUserHandler.UnLock,
	}
}

// MissedCron checks whether daily or monthly cron jobs were missed
// (for example if the service was down) and executes them manually.
//
// It ensures:
// - ExpireUsers runs once per day
// - ActiveMonthlyUsers runs once per month (on first day)
func (c *CornService) MissedCron() {
	db := database.GetConnection()

	state := stateManager.NewCronState()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	lastRun := state.DailyLastRun.Truncate(24 * time.Hour)

	// daily missed job
	logger.Info("Start checking missing daily cron jobs")
	if state.DailyLastRun.IsZero() || lastRun.Before(today) {
		logger.Info("Running missed DAILY cron...")
		c.ExpireUsers(context.Background(), db)
		c.DeleteExpiredUsers(context.Background(), db)
		state.DailyLastRun = today
	} else {
		logger.Info("Daily cron already ran today, skipping.")
	}
	logger.Info("Checking missing daily cron jobs completed")

	// monthly missed job
	logger.Info("start checking missing monthly cron jobs completed")
	firstDay := today.Day() == 1
	newMonth := state.MonthlyLastRun.IsZero() || state.MonthlyLastRun.Month() != today.Month()

	if firstDay && newMonth {
		logger.Info("Running missed MONTHLY cron...")
		c.ActiveMonthlyUsers(context.Background(), db)
		state.MonthlyLastRun = today
	}
	logger.Info("Checking missing monthly cron jobs completed")

	if err := state.Save(); err != nil {
		logger.Fatal("Failed to save state: %v", err)
	}
	logger.Info("Saving missing cron jobs completed")
}

// UserExpiryCron registers and starts all cron jobs:
//
// Daily (00:01:00):
//   - ExpireUsers
//
// Daily (00:02:00):
//   - DeleteExpiredUsers
//
// Monthly (1st & 2nd day at 00:01:00):
//   - ActiveMonthlyUsers
//
// The cron stops when context is canceled.
func (c *CornService) UserExpiryCron(ctx context.Context) {
	cronJob := cron.New(cron.WithSeconds())
	db := database.GetConnection()

	state := stateManager.NewCronState()

	// Every day at 00:01:00 — expire users
	_, err := cronJob.AddFunc("0 1 0 * * *", func() {
		c.ExpireUsers(ctx, db)

		state.DailyLastRun = time.Now().Truncate(24 * time.Hour)
		if err := state.Save(); err != nil {
			logger.Error("Failed to save state: %v", err)
		}
	})
	if err != nil {
		logger.Fatal("Failed to add cron job: %v", err)
	}
	logger.Info("Running user expiry cron...")

	// First and second day of each month at 00:01:00 — activate monthly users
	_, err = cronJob.AddFunc("0 1 0 1,2 * *", func() {
		c.ActiveMonthlyUsers(ctx, db)

		state.MonthlyLastRun = time.Now().Truncate(24 * time.Hour)
		if err = state.Save(); err != nil {
			logger.Error("Failed to update state: %v", err)
		}
	})
	if err != nil {
		logger.Fatal("Failed to add cron job: %v", err)
	}

	logger.Info("User activating Cron starting...")

	// Every day at 00:02:00 — delete expired users
	_, err3 := cronJob.AddFunc("0 2 0 * * *", func() {
		c.DeleteExpiredUsers(ctx, db)

		state.DailyLastRun = time.Now().Truncate(24 * time.Hour)
		if errSave := state.Save(); errSave != nil {
			logger.Error("Failed to save state: %v", errSave)
		}
	})
	if err3 != nil {
		logger.Fatal("Failed to add cron job: %v", err3)
	}
	logger.Info("Running delete expired users cron...")

	//// Test: run every minute at second 0
	//_, err = cronJob.AddFunc("0 * * * * *", func() {
	//	c.DeleteExpiredUsers(ctx, db)
	//})

	cronJob.Start()

	<-ctx.Done()
	logger.Warn("Received context cancel, shutting down...")
	cronJob.Stop()
	logger.Info("User activating Cron stopped...")
}

// ExpireUsers finds users whose expire_at has passed
// and deactivates them.
//
// Actions performed per user:
//   - Lock password and certificate authentication
//   - Disconnect active sessions
//   - Set deactivated_at = now and is_locked = true after enforcement succeeds
//
// Runs concurrently with max 10 workers.
func (c *CornService) ExpireUsers(
	ctx context.Context,
	db *gorm.DB,
) {
	var users []commonModels.OcservUser

	today := time.Now().
		UTC().
		Truncate(24 * time.Hour)

	err := db.WithContext(ctx).
		Select("id", "username", "expire_at").
		Where("expire_at IS NOT NULL").
		Where("expire_at < ?", today).
		Find(&users).
		Error

	if err != nil {
		logger.Error(
			"Failed to get users: %v",
			err,
		)

		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, u := range users {
		wg.Add(1)
		sem <- struct{}{}

		go func(u commonModels.OcservUser) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if err := c.enforceExpiredUser(
				u.Username,
			); err != nil {
				logger.Error(
					"Failed to enforce expiry for user %s: %v",
					u.Username,
					err,
				)

				return
			}

			now := time.Now().UTC()

			if err := db.WithContext(ctx).
				Model(&commonModels.OcservUser{}).
				Where("id = ?", u.ID).
				Updates(map[string]interface{}{
					"deactivated_at": now,
					"is_locked":      true,
				}).
				Error; err != nil {
				logger.Error(
					"Failed to update expired user %s: %v",
					u.Username,
					err,
				)
			}
		}(u)
	}

	wg.Wait()
}

func (c *CornService) enforceExpiredUser(
	username string,
) error {
	var enforcementErrors []error

	if _, err := c.lockUser(username); err != nil {
		enforcementErrors = append(
			enforcementErrors,
			fmt.Errorf(
				"lock authentication: %w",
				err,
			),
		)
	}

	if _, err := c.disconnectUser(username); err != nil {
		enforcementErrors = append(
			enforcementErrors,
			fmt.Errorf(
				"disconnect sessions: %w",
				err,
			),
		)
	}

	return errors.Join(enforcementErrors...)
}

// ActiveMonthlyUsers reactivates monthly traffic users
// at the beginning of a new month.
//
// Conditions:
//   - User is currently deactivated
//   - Traffic type is MonthlyReceive or MonthlyTransmit
//   - User is not expired
//
// Actions:
//   - Reset rx and tx counters
//   - Remove deactivated_at
//   - Unlock user
//
// Runs concurrently with max 10 workers.
func (c *CornService) ActiveMonthlyUsers(ctx context.Context, db *gorm.DB) {
	var users []commonModels.OcservUser
	today := time.Now().Truncate(24 * time.Hour)

	err := db.WithContext(ctx).
		Where("(expire_at IS NULL OR expire_at > ?)", today).
		Where("deactivated_at IS NOT NULL").
		Where("traffic_type IN ?", []string{
			commonModels.MonthlyReceive,
			commonModels.MonthlyTransmit,
			commonModels.MonthlyRxTx,
		}).
		Find(&users).Error
	if err != nil {
		logger.Error("Failed to get users: %v", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, u := range users {
		wg.Add(1)
		sem <- struct{}{}

		go func(u commonModels.OcservUser) {
			defer wg.Done()
			defer func() { <-sem }()

			now := time.Now()

			if err2 := db.Model(&u).Updates(map[string]interface{}{
				"rx":             0,
				"tx":             0,
				"usage_reset_at": &now,
				"deactivated_at": nil,
				"is_locked":      false,
			}).Error; err2 != nil {
				logger.Error("Failed to update user %s: %v", u.Username, err2)
				return
			}

			if _, err2 := c.unlockUser(
				u.Username,
			); err2 != nil {
				logger.Error(
					"Failed to unlock user %s: %v",
					u.Username,
					err2,
				)
			}

		}(u)
	}

	wg.Wait()
}

// DeleteExpiredUsers permanently deletes users who:
//
//   - Are deactivated
//   - Have been inactive longer than system.KeepInactiveUserDays
//   - AutoDeleteInactiveUsers setting is enabled
//
// Uses bulk delete for performance and logs number of deleted rows.
func (c *CornService) DeleteExpiredUsers(ctx context.Context, db *gorm.DB) {
	var system models.System
	err := db.WithContext(ctx).First(&system).Error
	if err != nil {
		logger.Error("Failed to get system: %v", err)
		logger.Warn("set KeepInactiveUserDays to `30` and AutoDeleteInactiveUsers to `false`")
		system.KeepInactiveUserDays = 30
		system.AutoDeleteInactiveUsers = false
	}

	if !system.AutoDeleteInactiveUsers {
		logger.Warn("User auto-delete is disabled")
		return
	}

	if system.KeepInactiveUserDays < 1 {
		logger.Warn("User keep inactive days is lower than 1 day")
		return
	}

	cutoffDate := time.Now().AddDate(0, 0, -system.KeepInactiveUserDays).UTC()
	result := db.WithContext(ctx).
		Where("expire_at IS NOT NULL AND expire_at <= ?", cutoffDate).
		Delete(&commonModels.OcservUser{})

	if result.Error != nil {
		logger.Error("Failed to delete inactive users: %v", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		logger.Info("No inactive users found for deletion")
		return
	}

	logger.Info("Deleted %d inactive users", result.RowsAffected)
}
