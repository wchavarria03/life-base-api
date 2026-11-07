package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const taskDateLayout = "2006-01-02"

// TaskLogRepository is the persistence boundary for tasks.task_logs.
type TaskLogRepository interface {
	List(ctx context.Context, filter models.TaskLogFilter) ([]*models.TaskLog, error)
	Create(ctx context.Context, input models.TaskLogInput) (*models.TaskLog, error)
	Delete(ctx context.Context, id string) error
}

// TaskLogService manages the home-maintenance history (tasks.task_logs).
type TaskLogService struct {
	logs TaskLogRepository
}

// NewTaskLogService constructs a TaskLogService.
func NewTaskLogService(logs TaskLogRepository) *TaskLogService {
	return &TaskLogService{logs: logs}
}

// Create records a manual "I did X" entry that needs no backing task.
func (s *TaskLogService) Create(ctx context.Context, input models.TaskLogInput) (*models.TaskLog, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.Category == "" {
		input.Category = models.TaskHouse
	}
	if err := validateTaskCategory(input.Category); err != nil {
		return nil, err
	}
	if input.PerformedAt != "" {
		if err := validateTaskDate("performed_at", input.PerformedAt); err != nil {
			return nil, err
		}
	}
	if input.Cost != nil && *input.Cost < 0 {
		return nil, fmt.Errorf("cost must be zero or greater")
	}
	input.Area = normalizeArea(input.Area)
	// Never trust a client-supplied owner: RLS requires user_id = auth.uid().
	input.UserID = userID
	return s.logs.Create(ctx, input)
}

// List returns logs matching filter, newest first. The result is never nil
// so it serializes as [] rather than null.
func (s *TaskLogService) List(ctx context.Context, filter models.TaskLogFilter) ([]*models.TaskLog, error) {
	if filter.From != "" {
		if err := validateTaskDate("from", filter.From); err != nil {
			return nil, err
		}
	}
	if filter.To != "" {
		if err := validateTaskDate("to", filter.To); err != nil {
			return nil, err
		}
	}
	if filter.Category != "" {
		if err := validateTaskCategory(models.TaskCategory(filter.Category)); err != nil {
			return nil, err
		}
	}
	logs, err := s.logs.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list task logs: %w", err)
	}
	if logs == nil {
		logs = []*models.TaskLog{}
	}
	return logs, nil
}

// Delete removes a log entry.
func (s *TaskLogService) Delete(ctx context.Context, id string) error {
	return s.logs.Delete(ctx, id)
}

// Summary aggregates a calendar year of logs: counts per month and per area,
// plus total cost per currency.
func (s *TaskLogService) Summary(ctx context.Context, year int) (*models.TaskLogSummary, error) {
	if year < 1970 || year > 9999 {
		return nil, fmt.Errorf("year is out of range")
	}
	logs, err := s.logs.List(ctx, models.TaskLogFilter{
		From: fmt.Sprintf("%04d-01-01", year),
		To:   fmt.Sprintf("%04d-12-31", year),
	})
	if err != nil {
		return nil, fmt.Errorf("list task logs: %w", err)
	}
	return summarizeTaskLogs(year, logs), nil
}

// summarizeTaskLogs is the pure aggregation behind Summary.
func summarizeTaskLogs(year int, logs []*models.TaskLog) *models.TaskLogSummary {
	summary := &models.TaskLogSummary{
		Year:           year,
		ByMonth:        make([]models.TaskLogMonthCount, 12),
		ByArea:         []models.TaskLogAreaSummary{},
		CostByCurrency: map[string]float64{},
	}
	for i := range summary.ByMonth {
		summary.ByMonth[i] = models.TaskLogMonthCount{Month: i + 1}
	}

	byArea := map[string]*models.TaskLogAreaSummary{}
	for _, l := range logs {
		summary.Total++

		if t, err := time.Parse(taskDateLayout, l.PerformedAt); err == nil {
			summary.ByMonth[int(t.Month())-1].Count++
		}

		area := ""
		if l.Area != nil {
			area = *l.Area
		}
		entry, ok := byArea[area]
		if !ok {
			entry = &models.TaskLogAreaSummary{Area: area, Costs: map[string]float64{}}
			byArea[area] = entry
		}
		entry.Count++

		if l.Cost != nil {
			currency := ""
			if l.Currency != nil {
				currency = *l.Currency
			}
			entry.Costs[currency] += *l.Cost
			summary.CostByCurrency[currency] += *l.Cost
		}
	}

	for _, entry := range byArea {
		summary.ByArea = append(summary.ByArea, *entry)
	}
	sort.Slice(summary.ByArea, func(i, j int) bool {
		if summary.ByArea[i].Count != summary.ByArea[j].Count {
			return summary.ByArea[i].Count > summary.ByArea[j].Count
		}
		return summary.ByArea[i].Area < summary.ByArea[j].Area
	})
	return summary
}

// WithLogs attaches the task-log repository so Complete can record history.
// Optional: with no repository attached, completing a task writes no log.
func (s *TaskService) WithLogs(logs TaskLogRepository) *TaskService {
	s.logs = logs
	return s
}

// recordCompletion writes the task_logs row for a completed task, snapshotting
// title/category/area. Best-effort: the task is already marked complete, so a
// failure here is logged rather than returned (retrying the whole completion
// would double-complete the task).
func (s *TaskService) recordCompletion(ctx context.Context, task *models.Task, opts models.TaskCompletion, performedAt string) {
	if s.logs == nil {
		return
	}
	taskID := task.ID
	_, err := s.logs.Create(ctx, models.TaskLogInput{
		UserID:      task.UserID,
		TaskID:      &taskID,
		Category:    task.Category,
		Title:       task.Title,
		Area:        task.Area,
		PerformedAt: performedAt,
		Note:        opts.Note,
		Cost:        opts.Cost,
		Currency:    opts.Currency,
	})
	if err != nil {
		log.Printf("task log: record completion of task=%s: %v", task.ID, err)
	}
}

// validateTaskCompletion checks the optional completion body.
func validateTaskCompletion(opts models.TaskCompletion) error {
	if opts.PerformedAt != "" {
		if err := validateTaskDate("performed_at", opts.PerformedAt); err != nil {
			return err
		}
	}
	if opts.Cost != nil && *opts.Cost < 0 {
		return fmt.Errorf("cost must be zero or greater")
	}
	return nil
}

func validateTaskCategory(c models.TaskCategory) error {
	switch c {
	case models.TaskHousehold, models.TaskHouse, models.TaskTodo:
		return nil
	}
	return fmt.Errorf("invalid category: %s", c)
}

func validateTaskDate(field, value string) error {
	if _, err := time.Parse(taskDateLayout, value); err != nil {
		return fmt.Errorf("%s must be a date in YYYY-MM-DD format", field)
	}
	return nil
}

// normalizeArea trims an area and turns blank into nil.
func normalizeArea(area *string) *string {
	if area == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*area)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// DigestTaskSource lists a user's open house/household tasks for the digest.
// Implemented by supabase.TaskRepository.
type DigestTaskSource interface {
	ListOpenByUser(ctx context.Context, userID string) ([]*models.Task, error)
}

// taskEffectiveDueDate returns the date a task is next due ("" if none),
// reconciling the legacy interval shape with due_date.
func taskEffectiveDueDate(t *models.Task) string {
	if t.RecurrenceType == nil || *t.RecurrenceType == "" {
		if t.IsRecurring {
			if t.LastCompletedDate == nil || t.IntervalDays == nil {
				return ""
			}
			completed, err := time.Parse(taskDateLayout, *t.LastCompletedDate)
			if err != nil {
				return ""
			}
			return completed.AddDate(0, 0, *t.IntervalDays).Format(taskDateLayout)
		}
	}
	if t.DueDate == nil {
		return ""
	}
	return *t.DueDate
}

// countTaskAlerts counts tasks that are overdue, and tasks that are due today
// or upcoming within their lead_time_days ("due soon"). today is YYYY-MM-DD.
func countTaskAlerts(tasks []*models.Task, today string) (overdue, dueSoon int) {
	todayT, err := time.Parse(taskDateLayout, today)
	if err != nil {
		return 0, 0
	}
	for _, t := range tasks {
		switch withTaskStatus(t).Status {
		case models.TaskOverdue:
			overdue++
		case models.TaskDueToday:
			dueSoon++
		case models.TaskUpcoming:
			if t.LeadTimeDays == nil {
				continue
			}
			due := taskEffectiveDueDate(t)
			if due == "" {
				continue
			}
			dueT, err := time.Parse(taskDateLayout, due)
			if err != nil {
				continue
			}
			if int(dueT.Sub(todayT).Hours()/24) <= *t.LeadTimeDays {
				dueSoon++
			}
		}
	}
	return overdue, dueSoon
}
