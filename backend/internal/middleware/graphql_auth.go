package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/auth"
	"github.com/kruakemaths/tru-activity/backend/pkg/permissions"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"gorm.io/gorm"
)

type GraphQLAuthMiddleware struct {
	jwtService  *auth.JWTService
	db          *gorm.DB
	permissions *permissions.PermissionChecker
}

type AuthContext struct {
	User        *models.User
	Claims      *auth.JWTClaims
	Permissions *permissions.PermissionChecker
}

const AuthContextKey = "auth"

func NewGraphQLAuthMiddleware(jwtService *auth.JWTService, db *gorm.DB) *GraphQLAuthMiddleware {
	return &GraphQLAuthMiddleware{
		jwtService:  jwtService,
		db:          db,
		permissions: permissions.NewPermissionChecker(),
	}
}

// ExtractAuth middleware สำหรับการ extract ข้อมูล auth จาก header
func (gam *GraphQLAuthMiddleware) ExtractAuth() graphql.HandlerExtension {
	return &authExtension{
		jwtService:  gam.jwtService,
		db:          gam.db,
		permissions: gam.permissions,
	}
}

type authExtension struct {
	jwtService  *auth.JWTService
	db          *gorm.DB
	permissions *permissions.PermissionChecker
}

func (ae *authExtension) ExtensionName() string {
	return "AuthExtension"
}

func (ae *authExtension) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (ae *authExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	// Extract token from context (HTTP headers)
	if reqCtx := graphql.GetOperationContext(ctx); reqCtx != nil {
		authHeader := reqCtx.Headers.Get("Authorization")
		if authHeader != "" {
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				if claims, err := ae.jwtService.ValidateToken(tokenParts[1]); err == nil {
					// Load user from database
					var user models.User
					if err := ae.db.Preload("Faculty").Preload("Department").First(&user, claims.UserID).Error; err == nil {
						authCtx := &AuthContext{
							User:        &user,
							Claims:      claims,
							Permissions: ae.permissions,
						}
						ctx = context.WithValue(ctx, AuthContextKey, authCtx)
					}
				}
			}
		}
	}

	return next(ctx)
}

func (ae *authExtension) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	return next(ctx)
}

func (ae *authExtension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return next(ctx)
}

// Helper functions สำหรับใช้ใน resolvers

// GetAuthContext ดึงข้อมูล auth จาก context
func GetAuthContext(ctx context.Context) (*AuthContext, error) {
	if authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext); ok {
		return authCtx, nil
	}
	return nil, fmt.Errorf("authentication required")
}

// RequireAuth ตรวจสอบว่า user ต้อง authenticate
func RequireAuth(ctx context.Context) (*AuthContext, error) {
	authCtx, err := GetAuthContext(ctx)
	if err != nil {
		return nil, gqlerror.Errorf("Authentication required")
	}
	return authCtx, nil
}

// RequireRole ตรวจสอบว่า user มี role ที่กำหนด
func RequireRole(ctx context.Context, roles ...models.UserRole) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if authCtx.User.Role == role {
			return authCtx, nil
		}
	}

	return nil, gqlerror.Errorf("Insufficient permissions")
}

// RequirePermission ตรวจสอบว่า user มี permission ที่กำหนด
func RequirePermission(ctx context.Context, permission permissions.Permission) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !authCtx.Permissions.HasPermission(authCtx.User, permission) {
		return nil, gqlerror.Errorf("Permission denied: %s", permission)
	}

	return authCtx, nil
}

// RequireFacultyPermission ตรวจสอบว่า user มี permission สำหรับ faculty
func RequireFacultyPermission(ctx context.Context, permission permissions.Permission, facultyID uint) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !authCtx.Permissions.HasFacultyPermission(authCtx.User, permission, facultyID) {
		return nil, gqlerror.Errorf("Faculty permission denied: %s for faculty %d", permission, facultyID)
	}

	return authCtx, nil
}

// RequireDepartmentPermission ตรวจสอบว่า user มี permission สำหรับ department
func RequireDepartmentPermission(ctx context.Context, permission permissions.Permission, departmentID uint) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !authCtx.Permissions.HasDepartmentPermission(authCtx.User, permission, departmentID) {
		return nil, gqlerror.Errorf("Department permission denied: %s for department %d", permission, departmentID)
	}

	return authCtx, nil
}

// CanManageUser ตรวจสอบว่าสามารถจัดการ user ได้หรือไม่
func CanManageUser(ctx context.Context, targetUserID uint, db *gorm.DB) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	var targetUser models.User
	if err := db.First(&targetUser, targetUserID).Error; err != nil {
		return nil, gqlerror.Errorf("User not found")
	}

	if !authCtx.Permissions.CanManageUser(authCtx.User, &targetUser) {
		return nil, gqlerror.Errorf("Cannot manage this user")
	}

	return authCtx, nil
}

// IsOwnerOrAdmin ตรวจสอบว่าเป็นเจ้าของหรือ admin
func IsOwnerOrAdmin(ctx context.Context, ownerID uint) (*AuthContext, error) {
	authCtx, err := RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// ถ้าเป็นเจ้าของหรือเป็น admin
	if authCtx.User.ID == ownerID || authCtx.User.IsAdmin() {
		return authCtx, nil
	}

	return nil, gqlerror.Errorf("Access denied: not owner or admin")
}

// FilterByFaculty กรองข้อมูลตาม faculty ของ user
func FilterByFaculty(ctx context.Context, query *gorm.DB, facultyField string) *gorm.DB {
	authCtx, err := GetAuthContext(ctx)
	if err != nil {
		return query
	}

	// Super admin สามารถดูทุกอย่าง
	if authCtx.User.Role == models.UserRoleSuperAdmin {
		return query
	}

	// Faculty admin และ regular admin เห็นเฉพาะของคณะตัวเอง
	if authCtx.User.FacultyID != nil {
		if facultyField != "" {
			return query.Where(facultyField+" = ? OR "+facultyField+" IS NULL", *authCtx.User.FacultyID)
		}
	}

	return query
}