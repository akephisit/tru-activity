package graph

import (
	"github.com/kruakemaths/tru-activity/backend/internal/database"
	"github.com/kruakemaths/tru-activity/backend/pkg/auth"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	DB         *database.DB
	JWTService *auth.JWTService
}