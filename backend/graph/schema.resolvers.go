package graph

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kruakemaths/tru-activity/backend/graph/generated"
	"github.com/kruakemaths/tru-activity/backend/graph/model"
	"github.com/kruakemaths/tru-activity/backend/internal/middleware"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/permissions"
	"github.com/kruakemaths/tru-activity/backend/pkg/utils"
)

// Authentication Resolvers

func (r *mutationResolver) Login(ctx context.Context, input model.LoginInput) (*model.AuthPayload, error) {
	var user models.User
	if err := r.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate JWT token with faculty and department info
	token, err := r.JWTService.GenerateToken(
		user.ID, 
		user.Email, 
		string(user.Role),
		user.FacultyID,
		user.DepartmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token")
	}

	// Update last login
	r.DB.Model(&user).Update("last_login_at", "NOW()")

	return &model.AuthPayload{
		Token: token,
		User:  convertUserToGraphQL(&user),
	}, nil
}

func (r *mutationResolver) Register(ctx context.Context, input model.RegisterInput) (*model.AuthPayload, error) {
	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	// Generate QR secret
	qrSecret := utils.GenerateQRSecret()

	var facultyID, departmentID *uint
	if input.FacultyID != nil {
		id, _ := strconv.ParseUint(*input.FacultyID, 10, 32)
		facultyIDUint := uint(id)
		facultyID = &facultyIDUint
	}
	if input.DepartmentID != nil {
		id, _ := strconv.ParseUint(*input.DepartmentID, 10, 32)
		departmentIDUint := uint(id)
		departmentID = &departmentIDUint
	}

	user := models.User{
		StudentID:    input.StudentID,
		Email:        input.Email,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Password:     hashedPassword,
		Role:         models.UserRoleStudent,
		QRSecret:     qrSecret,
		FacultyID:    facultyID,
		DepartmentID: departmentID,
		IsActive:     true,
	}

	if err := r.DB.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user")
	}

	// Generate JWT token
	token, err := r.JWTService.GenerateToken(
		user.ID,
		user.Email,
		string(user.Role),
		user.FacultyID,
		user.DepartmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token")
	}

	return &model.AuthPayload{
		Token: token,
		User:  convertUserToGraphQL(&user),
	}, nil
}

func (r *mutationResolver) RefreshToken(ctx context.Context) (*model.AuthPayload, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Generate new token
	token, err := r.JWTService.GenerateToken(
		authCtx.User.ID,
		authCtx.User.Email,
		string(authCtx.User.Role),
		authCtx.User.FacultyID,
		authCtx.User.DepartmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token")
	}

	return &model.AuthPayload{
		Token: token,
		User:  convertUserToGraphQL(authCtx.User),
	}, nil
}

// Query Resolvers

func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	return convertUserToGraphQL(authCtx.User), nil
}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int) ([]*model.User, error) {
	_, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin, models.UserRoleRegularAdmin)
	if err != nil {
		return nil, err
	}

	query := r.DB.Model(&models.User{}).Preload("Faculty").Preload("Department")
	
	// Apply faculty filtering for non-super admins
	query = middleware.FilterByFaculty(ctx, query, "faculty_id")

	if offset != nil {
		query = query.Offset(*offset)
	}
	if limit != nil {
		query = query.Limit(*limit)
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch users")
	}

	result := make([]*model.User, len(users))
	for i, user := range users {
		result[i] = convertUserToGraphQL(&user)
	}
	return result, nil
}

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	var user models.User
	if err := r.DB.Preload("Faculty").Preload("Department").First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check permission to view user
	if !authCtx.User.CanViewUser(&user) {
		return nil, fmt.Errorf("permission denied")
	}

	return convertUserToGraphQL(&user), nil
}

func (r *queryResolver) Faculties(ctx context.Context) ([]*model.Faculty, error) {
	_, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	var faculties []models.Faculty
	if err := r.DB.Where("is_active = ?", true).Find(&faculties).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch faculties")
	}

	result := make([]*model.Faculty, len(faculties))
	for i, faculty := range faculties {
		result[i] = convertFacultyToGraphQL(&faculty)
	}
	return result, nil
}

func (r *queryResolver) Faculty(ctx context.Context, id string) (*model.Faculty, error) {
	_, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	facultyID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid faculty ID")
	}

	var faculty models.Faculty
	if err := r.DB.First(&faculty, facultyID).Error; err != nil {
		return nil, fmt.Errorf("faculty not found")
	}

	return convertFacultyToGraphQL(&faculty), nil
}

func (r *queryResolver) Activities(ctx context.Context, limit *int, offset *int, facultyID *string, status *model.ActivityStatus) ([]*model.Activity, error) {
	_, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	query := r.DB.Model(&models.Activity{}).Preload("Faculty").Preload("Department").Preload("CreatedBy")

	// Apply faculty filtering
	query = middleware.FilterByFaculty(ctx, query, "faculty_id")

	if facultyID != nil {
		fID, _ := strconv.ParseUint(*facultyID, 10, 32)
		query = query.Where("faculty_id = ?", fID)
	}

	if status != nil {
		query = query.Where("status = ?", string(*status))
	}

	if offset != nil {
		query = query.Offset(*offset)
	}
	if limit != nil {
		query = query.Limit(*limit)
	}

	var activities []models.Activity
	if err := query.Find(&activities).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch activities")
	}

	result := make([]*model.Activity, len(activities))
	for i, activity := range activities {
		result[i] = convertActivityToGraphQL(&activity)
	}
	return result, nil
}

// Activity Management Mutations

func (r *mutationResolver) CreateActivity(ctx context.Context, input model.CreateActivityInput) (*model.Activity, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin)
	if err != nil {
		return nil, err
	}

	var facultyID, departmentID *uint
	if input.FacultyID != nil {
		id, _ := strconv.ParseUint(*input.FacultyID, 10, 32)
		facultyIDUint := uint(id)
		facultyID = &facultyIDUint

		// Check faculty permission
		if !authCtx.Permissions.HasFacultyPermission(authCtx.User, permissions.PermCreateActivity, facultyIDUint) {
			return nil, fmt.Errorf("permission denied for this faculty")
		}
	}

	if input.DepartmentID != nil {
		id, _ := strconv.ParseUint(*input.DepartmentID, 10, 32)
		departmentIDUint := uint(id)
		departmentID = &departmentIDUint
	}

	activity := models.Activity{
		Title:           input.Title,
		Description:     input.Description,
		Type:            models.ActivityType(input.Type),
		Status:          models.ActivityStatusDraft,
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		Location:        input.Location,
		MaxParticipants: input.MaxParticipants,
		RequireApproval: input.RequireApproval,
		Points:          input.Points,
		FacultyID:       facultyID,
		DepartmentID:    departmentID,
		CreatedByID:     authCtx.User.ID,
	}

	if err := r.DB.Create(&activity).Error; err != nil {
		return nil, fmt.Errorf("failed to create activity")
	}

	// Load relationships
	r.DB.Preload("Faculty").Preload("Department").Preload("CreatedBy").First(&activity, activity.ID)

	return convertActivityToGraphQL(&activity), nil
}

// Faculty Management Mutations

func (r *mutationResolver) CreateFaculty(ctx context.Context, input model.CreateFacultyInput) (*model.Faculty, error) {
	_, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin)
	if err != nil {
		return nil, err
	}

	faculty := models.Faculty{
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
		IsActive:    true,
	}

	if err := r.DB.Create(&faculty).Error; err != nil {
		return nil, fmt.Errorf("failed to create faculty")
	}

	return convertFacultyToGraphQL(&faculty), nil
}

// Participation Management

func (r *mutationResolver) JoinActivity(ctx context.Context, activityID string) (*model.Participation, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	actID, err := strconv.ParseUint(activityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID")
	}

	// Check if activity exists and is active
	var activity models.Activity
	if err := r.DB.First(&activity, actID).Error; err != nil {
		return nil, fmt.Errorf("activity not found")
	}

	if activity.Status != models.ActivityStatusActive {
		return nil, fmt.Errorf("activity is not active")
	}

	// Check if already participating
	var existingParticipation models.Participation
	if err := r.DB.Where("user_id = ? AND activity_id = ?", authCtx.User.ID, actID).First(&existingParticipation).Error; err == nil {
		return nil, fmt.Errorf("already participating in this activity")
	}

	status := models.ParticipationStatusApproved
	if activity.RequireApproval {
		status = models.ParticipationStatusPending
	}

	participation := models.Participation{
		UserID:     authCtx.User.ID,
		ActivityID: uint(actID),
		Status:     status,
	}

	if err := r.DB.Create(&participation).Error; err != nil {
		return nil, fmt.Errorf("failed to join activity")
	}

	// Load relationships
	r.DB.Preload("User").Preload("Activity").First(&participation, participation.ID)

	return convertParticipationToGraphQL(&participation), nil
}

// Helper conversion functions

func convertUserToGraphQL(user *models.User) *model.User {
	return &model.User{
		ID:           fmt.Sprintf("%d", user.ID),
		StudentID:    user.StudentID,
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:         model.UserRole(user.Role),
		QrSecret:     user.QRSecret,
		IsActive:     user.IsActive,
		LastLoginAt:  user.LastLoginAt,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func convertFacultyToGraphQL(faculty *models.Faculty) *model.Faculty {
	return &model.Faculty{
		ID:          fmt.Sprintf("%d", faculty.ID),
		Name:        faculty.Name,
		Code:        faculty.Code,
		Description: faculty.Description,
		IsActive:    faculty.IsActive,
		CreatedAt:   faculty.CreatedAt,
		UpdatedAt:   faculty.UpdatedAt,
	}
}

func convertActivityToGraphQL(activity *models.Activity) *model.Activity {
	return &model.Activity{
		ID:              fmt.Sprintf("%d", activity.ID),
		Title:           activity.Title,
		Description:     activity.Description,
		Type:            model.ActivityType(activity.Type),
		Status:          model.ActivityStatus(activity.Status),
		StartDate:       activity.StartDate,
		EndDate:         activity.EndDate,
		Location:        activity.Location,
		MaxParticipants: activity.MaxParticipants,
		RequireApproval: activity.RequireApproval,
		Points:          activity.Points,
		CreatedAt:       activity.CreatedAt,
		UpdatedAt:       activity.UpdatedAt,
	}
}

func convertParticipationToGraphQL(participation *models.Participation) *model.Participation {
	return &model.Participation{
		ID:           fmt.Sprintf("%d", participation.ID),
		Status:       model.ParticipationStatus(participation.Status),
		RegisteredAt: participation.RegisteredAt,
		ApprovedAt:   participation.ApprovedAt,
		AttendedAt:   participation.AttendedAt,
		Notes:        participation.Notes,
		CreatedAt:    participation.CreatedAt,
		UpdatedAt:    participation.UpdatedAt,
	}
}

// Resolver type definitions
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }