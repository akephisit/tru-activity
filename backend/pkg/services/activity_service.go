package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"gorm.io/gorm"
)

type ActivityService struct {
	DB *gorm.DB
}

type RecurrenceRule struct {
	Frequency string    // DAILY, WEEKLY, MONTHLY
	Interval  int       // Every N days/weeks/months
	Count     int       // Number of occurrences (-1 for infinite)
	Until     *time.Time // End date
	ByWeekDay []int     // Days of week (0=Sunday, 1=Monday, etc.)
}

func NewActivityService(db *gorm.DB) *ActivityService {
	return &ActivityService{DB: db}
}

// CreateActivityFromTemplate creates a new activity from a template
func (as *ActivityService) CreateActivityFromTemplate(templateID uint, input *ActivityInput, createdByID uint) (*models.Activity, error) {
	var template models.ActivityTemplate
	if err := as.DB.First(&template, templateID).Error; err != nil {
		return nil, fmt.Errorf("template not found: %v", err)
	}

	// Create activity with template defaults
	activity := models.Activity{
		Title:           input.Title,
		Description:     input.Description,
		Type:            template.Type,
		Status:          models.ActivityStatusDraft,
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		Location:        template.Location,
		MaxParticipants: template.MaxParticipants,
		RequireApproval: template.RequireApproval,
		Points:          template.Points,
		QRCodeRequired:  template.QRCodeRequired,
		AutoApprove:     template.AutoApprove,
		TemplateID:      &templateID,
		FacultyID:       template.FacultyID,
		CreatedByID:     createdByID,
	}

	// Override with input values if provided
	if input.Location != "" {
		activity.Location = input.Location
	}
	if input.MaxParticipants != nil {
		activity.MaxParticipants = input.MaxParticipants
	}
	if input.RequireApproval != nil {
		activity.RequireApproval = *input.RequireApproval
	}
	if input.Points != nil {
		activity.Points = *input.Points
	}
	if input.QRCodeRequired != nil {
		activity.QRCodeRequired = *input.QRCodeRequired
	}
	if input.AutoApprove != nil {
		activity.AutoApprove = *input.AutoApprove
	}
	if input.FacultyID != nil {
		activity.FacultyID = input.FacultyID
	}
	if input.DepartmentID != nil {
		activity.DepartmentID = input.DepartmentID
	}

	if err := as.DB.Create(&activity).Error; err != nil {
		return nil, fmt.Errorf("failed to create activity: %v", err)
	}

	return &activity, nil
}

// CreateRecurringActivities creates multiple activities based on recurrence rule
func (as *ActivityService) CreateRecurringActivities(baseActivity *models.Activity, recurrenceRule string, createdByID uint) ([]*models.Activity, error) {
	rule, err := as.parseRecurrenceRule(recurrenceRule)
	if err != nil {
		return nil, fmt.Errorf("invalid recurrence rule: %v", err)
	}

	activities := []*models.Activity{}
	currentDate := baseActivity.StartDate
	duration := baseActivity.EndDate.Sub(baseActivity.StartDate)
	
	// Create parent activity
	baseActivity.IsRecurring = true
	baseActivity.RecurrenceRule = recurrenceRule
	if err := as.DB.Create(baseActivity).Error; err != nil {
		return nil, fmt.Errorf("failed to create base activity: %v", err)
	}
	activities = append(activities, baseActivity)

	// Generate recurring activities
	for i := 0; i < rule.Count || (rule.Count == -1 && i < 100); i++ { // Limit to 100 if infinite
		if rule.Until != nil && currentDate.After(*rule.Until) {
			break
		}

		nextDate := as.calculateNextDate(currentDate, rule, i+1)
		if nextDate.Equal(currentDate) {
			break // Avoid infinite loop
		}

		childActivity := *baseActivity // Copy base activity
		childActivity.ID = 0          // Reset ID for new record
		childActivity.StartDate = nextDate
		childActivity.EndDate = nextDate.Add(duration)
		childActivity.ParentActivityID = &baseActivity.ID
		childActivity.IsRecurring = false
		childActivity.RecurrenceRule = ""
		childActivity.Title = fmt.Sprintf("%s (%s)", baseActivity.Title, nextDate.Format("2006-01-02"))

		if err := as.DB.Create(&childActivity).Error; err != nil {
			return activities, fmt.Errorf("failed to create recurring activity: %v", err)
		}

		activities = append(activities, &childActivity)
		currentDate = nextDate
	}

	return activities, nil
}

// AssignActivityToAdmin assigns an activity to a regular admin
func (as *ActivityService) AssignActivityToAdmin(activityID, adminID, assignedByID uint, permissions ActivityAssignmentPermissions) (*models.ActivityAssignment, error) {
	// Verify admin is Regular Admin
	var admin models.User
	if err := as.DB.First(&admin, adminID).Error; err != nil {
		return nil, fmt.Errorf("admin not found: %v", err)
	}

	if admin.Role != models.UserRoleRegularAdmin {
		return nil, fmt.Errorf("user is not a regular admin")
	}

	// Check if assignment already exists
	var existingAssignment models.ActivityAssignment
	if err := as.DB.Where("activity_id = ? AND admin_id = ?", activityID, adminID).First(&existingAssignment).Error; err == nil {
		return nil, fmt.Errorf("activity already assigned to this admin")
	}

	assignment := models.ActivityAssignment{
		ActivityID:   activityID,
		AdminID:      adminID,
		AssignedByID: assignedByID,
		CanScanQR:    permissions.CanScanQR,
		CanApprove:   permissions.CanApprove,
		Notes:        permissions.Notes,
	}

	if err := as.DB.Create(&assignment).Error; err != nil {
		return nil, fmt.Errorf("failed to create assignment: %v", err)
	}

	// Load associations
	as.DB.Preload("Activity").Preload("Admin").Preload("AssignedBy").First(&assignment, assignment.ID)

	return &assignment, nil
}

// GetAdminActivities gets activities assigned to a regular admin
func (as *ActivityService) GetAdminActivities(adminID uint) ([]models.Activity, error) {
	var assignments []models.ActivityAssignment
	if err := as.DB.Preload("Activity").Where("admin_id = ?", adminID).Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %v", err)
	}

	activities := make([]models.Activity, len(assignments))
	for i, assignment := range assignments {
		activities[i] = assignment.Activity
	}

	return activities, nil
}

// GetFacultyActivities gets activities scoped to a faculty
func (as *ActivityService) GetFacultyActivities(facultyID uint, includeAllFaculty bool) ([]models.Activity, error) {
	query := as.DB.Preload("Faculty").Preload("Department").Preload("CreatedBy")

	if includeAllFaculty {
		// Include both faculty-specific and cross-faculty activities
		query = query.Where("faculty_id = ? OR faculty_id IS NULL", facultyID)
	} else {
		query = query.Where("faculty_id = ?", facultyID)
	}

	var activities []models.Activity
	if err := query.Order("start_date DESC").Find(&activities).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch faculty activities: %v", err)
	}

	return activities, nil
}

// Helper types and methods

type ActivityInput struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	StartDate       time.Time  `json:"start_date"`
	EndDate         time.Time  `json:"end_date"`
	Location        string     `json:"location"`
	MaxParticipants *int       `json:"max_participants"`
	RequireApproval *bool      `json:"require_approval"`
	Points          *int       `json:"points"`
	QRCodeRequired  *bool      `json:"qr_code_required"`
	AutoApprove     *bool      `json:"auto_approve"`
	FacultyID       *uint      `json:"faculty_id"`
	DepartmentID    *uint      `json:"department_id"`
}

type ActivityAssignmentPermissions struct {
	CanScanQR  bool   `json:"can_scan_qr"`
	CanApprove bool   `json:"can_approve"`
	Notes      string `json:"notes"`
}

func (as *ActivityService) parseRecurrenceRule(rule string) (*RecurrenceRule, error) {
	// Simple RRULE parser for basic recurrence patterns
	// Format: FREQ=WEEKLY;INTERVAL=1;COUNT=10;BYDAY=MO,WE,FR
	
	parts := strings.Split(rule, ";")
	rr := &RecurrenceRule{
		Frequency: "WEEKLY",
		Interval:  1,
		Count:     -1, // Infinite by default
	}

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}

		key, value := kv[0], kv[1]
		switch key {
		case "FREQ":
			rr.Frequency = value
		case "INTERVAL":
			if interval, err := strconv.Atoi(value); err == nil {
				rr.Interval = interval
			}
		case "COUNT":
			if count, err := strconv.Atoi(value); err == nil {
				rr.Count = count
			}
		case "UNTIL":
			if until, err := time.Parse("20060102T150405Z", value); err == nil {
				rr.Until = &until
			}
		case "BYDAY":
			days := strings.Split(value, ",")
			for _, day := range days {
				switch day {
				case "SU":
					rr.ByWeekDay = append(rr.ByWeekDay, 0)
				case "MO":
					rr.ByWeekDay = append(rr.ByWeekDay, 1)
				case "TU":
					rr.ByWeekDay = append(rr.ByWeekDay, 2)
				case "WE":
					rr.ByWeekDay = append(rr.ByWeekDay, 3)
				case "TH":
					rr.ByWeekDay = append(rr.ByWeekDay, 4)
				case "FR":
					rr.ByWeekDay = append(rr.ByWeekDay, 5)
				case "SA":
					rr.ByWeekDay = append(rr.ByWeekDay, 6)
				}
			}
		}
	}

	return rr, nil
}

func (as *ActivityService) calculateNextDate(baseDate time.Time, rule *RecurrenceRule, occurrence int) time.Time {
	switch rule.Frequency {
	case "DAILY":
		return baseDate.AddDate(0, 0, rule.Interval*occurrence)
	case "WEEKLY":
		if len(rule.ByWeekDay) > 0 {
			// Find next occurrence on specified weekdays
			return as.findNextWeekdayOccurrence(baseDate, rule.ByWeekDay, occurrence)
		}
		return baseDate.AddDate(0, 0, 7*rule.Interval*occurrence)
	case "MONTHLY":
		return baseDate.AddDate(0, rule.Interval*occurrence, 0)
	default:
		return baseDate.AddDate(0, 0, 7*occurrence) // Default to weekly
	}
}

func (as *ActivityService) findNextWeekdayOccurrence(baseDate time.Time, weekdays []int, occurrence int) time.Time {
	// Simple implementation - find next occurrence in weekdays
	current := baseDate
	count := 0
	
	for count < occurrence {
		current = current.AddDate(0, 0, 1)
		weekday := int(current.Weekday())
		
		for _, wd := range weekdays {
			if weekday == wd {
				count++
				if count == occurrence {
					return current
				}
				break
			}
		}
	}
	
	return current
}

// ValidateRecurrenceRule validates a recurrence rule string
func (as *ActivityService) ValidateRecurrenceRule(rule string) error {
	if rule == "" {
		return nil // Empty rule is valid (no recurrence)
	}

	// Basic validation using regex
	validPattern := regexp.MustCompile(`^FREQ=(DAILY|WEEKLY|MONTHLY)(;INTERVAL=\d+)?(;COUNT=\d+)?(;UNTIL=\d{8}T\d{6}Z)?(;BYDAY=(SU|MO|TU|WE|TH|FR|SA)(,(SU|MO|TU|WE|TH|FR|SA))*)?$`)
	
	if !validPattern.MatchString(rule) {
		return fmt.Errorf("invalid recurrence rule format")
	}

	return nil
}