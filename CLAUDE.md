# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**TRU Activity** - ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี

Full-stack web application ที่พัฒนาด้วย Go Fiber + GraphQL + PostgreSQL + SvelteKit สำหรับจัดการกิจกรรมมหาวิทยาลัย พร้อม multi-level admin system และ role-based access control

## Project Structure

```
/
├── backend/                 # Go Fiber + GraphQL Backend
│   ├── cmd/server/         # Application entry point
│   ├── internal/           # Internal packages
│   │   ├── config/        # Configuration management
│   │   ├── database/      # Database connection & migrations
│   │   ├── middleware/    # HTTP middleware (auth, CORS)
│   │   ├── models/        # GORM database models
│   │   └── handlers/      # HTTP handlers
│   ├── pkg/               # Public packages
│   │   ├── auth/          # JWT authentication service
│   │   └── utils/         # Utility functions (password, QR)
│   ├── graph/             # GraphQL schema และ resolvers
│   ├── migrations/        # PostgreSQL database migrations
│   └── Dockerfile         # Backend container config
├── frontend/              # SvelteKit Frontend
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/  # Svelte components + shadcn-ui
│   │   │   ├── graphql/     # Apollo Client + GraphQL operations
│   │   │   ├── stores/      # Svelte stores (auth, etc.)
│   │   │   └── generated/   # Generated GraphQL types
│   │   └── routes/          # SvelteKit routes (auth, dashboard)
│   └── Dockerfile.dev      # Frontend container config
├── docker-compose.yml     # Development environment
├── scripts/               # Development automation scripts
└── docs/                  # Project documentation
```

### Key Directories

**Backend:**

- `backend/cmd/server/` - Main application entry point
- `backend/internal/models/` - Database models (User, Activity, Faculty, etc.)
- `backend/graph/` - GraphQL schema definitions และ resolvers
- `backend/pkg/auth/` - JWT authentication และ authorization logic
- `backend/migrations/` - Database migration scripts

**Frontend:**

- `frontend/src/routes/` - SvelteKit routes (login, register, dashboard)
- `frontend/src/lib/components/` - Reusable Svelte components
- `frontend/src/lib/components/ui/` - shadcn-svelte UI component library
- `frontend/src/lib/graphql/` - GraphQL queries, mutations และ Apollo Client setup
- `frontend/src/lib/stores/` - Svelte stores สำหรับ state management

## Common Development Commands

### Quick Start (Recommended)

```bash
# Start entire development environment
./scripts/dev-start.sh

# หรือใช้ Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Frontend Development

Commands should be run from the `frontend/` directory:

```bash
cd frontend

# Development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npm run check

# Type checking with watch mode
npm run check:watch

# Linting
npm run lint

# Generate GraphQL types
npm run codegen

# Generate GraphQL types with watch mode
npm run codegen:watch
```

### Backend Development

Commands should be run from the `backend/` directory:

```bash
cd backend

# Install dependencies
go mod tidy

# Run with hot reload (requires air)
air

# Run normally
go run cmd/server/main.go

# Build binary
go build -o main cmd/server/main.go

# Run tests
go test ./...

# Generate GraphQL code
go run github.com/99designs/gqlgen generate
```

### Database Operations

```bash
# Access PostgreSQL container
docker exec -it tru-activity-db psql -U postgres -d tru_activity

# Run migrations
docker exec tru-activity-db psql -U postgres -d tru_activity -f /docker-entrypoint-initdb.d/001_initial_schema.sql

# Backup database
docker exec tru-activity-db pg_dump -U postgres tru_activity > backup.sql

# Redis CLI
docker exec -it tru-activity-redis redis-cli
```

## Architecture Overview

### Backend Technology Stack

- **Go 1.22+**: Primary programming language
- **Fiber v2**: High-performance HTTP web framework
- **GraphQL**: API query language using gqlgen
- **PostgreSQL**: Primary database with GORM ORM
- **Redis**: Caching และ session storage
- **JWT**: Authentication tokens
- **Docker**: Containerization และ development environment

### Frontend Technology Stack

- **SvelteKit**: Full-stack framework with file-based routing
- **Svelte 5**: Component framework with runes syntax
- **TypeScript**: Type-safe JavaScript development
- **TailwindCSS**: Utility-first CSS framework
- **shadcn-svelte**: UI component library (bits-ui based)
- **Apollo Client**: GraphQL client with caching
- **Zod**: Runtime type validation

### Data Management

- **GraphQL**: API layer with queries, mutations และ subscriptions
- **Apollo Client**: Frontend GraphQL client with normalized caching
- **GORM**: Go ORM สำหรับ database operations
- **PostgreSQL**: Relational database with proper indexing
- **Redis**: Caching layer และ session management

### Key Features

- **Authentication**: JWT-based authentication with refresh tokens
- **Authorization**: Role-based access control (Student, Regular Admin, Faculty Admin, Super Admin)
- **Activity Management**: Create, manage และ track university activities
- **Participation System**: Registration, approval และ attendance tracking
- **QR Integration**: QR secret keys สำหรับ users
- **Subscription Tracking**: Monitor subscription expiry
- **Real-time Updates**: GraphQL subscriptions สำหรับ live data

### Routing Structure

**Frontend Routes:**

- `/` - Landing page
- `/login` - Authentication page
- `/register` - User registration with faculty/department selection
- `/dashboard` - Main dashboard (role-based content)
- `/dashboard/activities` - Activity listing และ management
- `/dashboard/my-activities` - User's activities (students)
- `/dashboard/manage-activities` - Activity management (admins)
- `/dashboard/users` - User management (admins)
- `/dashboard/faculties` - Faculty management (super admin)
- `/dashboard/departments` - Department management (admins)
- `/dashboard/reports` - Analytics และ reports (admins)

**Backend API:**

- `/health` - Health check endpoint
- `/query` - GraphQL endpoint (POST)
- `/` - GraphQL Playground (development only)

### Development Patterns

- **Backend**: Clean architecture with internal packages, dependency injection
- **Frontend**: Component-based architecture with Svelte 5 runes syntax
- **TypeScript**: Strict typing throughout both frontend และ backend (via GraphQL codegen)
- **GraphQL First**: API-first development with schema-driven approach
- **Container-based**: Docker Compose สำหรับ consistent development environment

## Database Schema

The database schema is defined in `backend/internal/models/` และ `backend/migrations/001_initial_schema.sql`:

### Core Tables

**Users** (`users`)

- `id`: Primary key
- `student_id`: Unique student identifier
- `email`: User email (unique)
- `first_name`, `last_name`: User names
- `password`: Hashed password
- `role`: User role (student, super_admin, faculty_admin, regular_admin)
- `qr_secret`: QR code secret key
- `faculty_id`, `department_id`: Foreign keys to faculty/department
- `is_active`: Account status
- `last_login_at`: Last login timestamp

**Faculties** (`faculties`)

- `id`: Primary key
- `name`: Faculty name
- `code`: Unique faculty code
- `description`: Faculty description
- `is_active`: Status flag

**Departments** (`departments`)

- `id`: Primary key
- `name`: Department name
- `code`: Department code (unique within faculty)
- `faculty_id`: Foreign key to faculty
- `is_active`: Status flag

**Activities** (`activities`)

- `id`: Primary key
- `title`: Activity title
- `description`: Activity description
- `type`: Activity type (workshop, seminar, competition, volunteer, other)
- `status`: Activity status (draft, active, completed, cancelled)
- `start_date`, `end_date`: Activity dates
- `location`: Activity location
- `max_participants`: Maximum participants limit
- `require_approval`: Whether registration requires approval
- `points`: Points awarded for participation
- `faculty_id`, `department_id`: Optional faculty/department association
- `created_by_id`: User who created the activity

**Participations** (`participations`)

- `id`: Primary key
- `user_id`: Foreign key to user
- `activity_id`: Foreign key to activity
- `status`: Participation status (pending, approved, rejected, attended, absent)
- `registered_at`: Registration timestamp
- `approved_at`: Approval timestamp
- `attended_at`: Attendance timestamp
- `notes`: Additional notes

**Subscriptions** (`subscriptions`)

- `id`: Primary key
- `user_id`: Foreign key to user
- `type`: Subscription type (basic, premium, vip)
- `status`: Subscription status (active, expired, cancelled)
- `start_date`, `end_date`: Subscription period

### Relationships

- Users belong to Faculty และ Department (optional)
- Activities can be associated with Faculty และ Department
- Participations link Users และ Activities (many-to-many)
- Subscriptions belong to Users
- All tables have proper foreign key constraints และ indexes

## GraphQL API

The GraphQL API is defined in `backend/graph/schema.graphqls` และ provides:

### Core Operations

**Queries:**

- `me`: Get current user info
- `users`: List users (with pagination)
- `faculties`: List all faculties
- `departments`: List departments (optionally filtered by faculty)
- `activities`: List activities (with filters และ pagination)
- `myActivities`: Get activities created by current user
- `myParticipations`: Get current user's participations

**Mutations:**

- `login`, `register`: Authentication operations
- `createActivity`, `updateActivity`, `deleteActivity`: Activity management
- `joinActivity`, `leaveActivity`: Participation management
- `approveParticipation`, `rejectParticipation`: Admin approval operations
- `markAttendance`: Attendance tracking
- `createFaculty`, `updateFaculty`, `deleteFaculty`: Faculty management (super admin)
- `createDepartment`, `updateDepartment`, `deleteDepartment`: Department management

**Subscriptions:**

- `activityUpdated`: Real-time activity updates
- `participationUpdated`: Real-time participation updates

### Access Control

- **Public**: `login`, `register`
- **Authenticated**: `me`, basic queries
- **Student**: Activity viewing และ participation
- **Admin**: Activity และ user management based on role level
- **Super Admin**: Full system access

## Documentation and References

### Backend Documentation

- **Go**: https://golang.org/doc/
- **Fiber**: https://docs.gofiber.io/
- **GORM**: https://gorm.io/docs/
- **gqlgen**: https://gqlgen.com/getting-started/
- **PostgreSQL**: https://www.postgresql.org/docs/

### Frontend Documentation

- **SvelteKit**: https://kit.svelte.dev/docs
- **Svelte 5**: https://svelte.dev/docs/svelte/overview
- **TailwindCSS**: https://tailwindcss.com/docs
- **shadcn-svelte**: https://www.shadcn-svelte.com/docs
- **Apollo Client**: https://www.apollographql.com/docs/react/
- **Zod**: https://zod.dev/

## Development Guidelines

### Code Style และ Best Practices

**Backend (Go):**

- ใช้ Go standard formatting (`gofmt`)
- Follow clean architecture principles
- Use dependency injection pattern
- Proper error handling with context
- Use GORM best practices สำหรับ database operations
- JWT tokens should be properly validated และ refreshed

**Frontend (SvelteKit):**

- ใช้ TypeScript throughout
- Follow Svelte 5 runes syntax (`$props()`, `$state()`, `$derived()`)
- Use shadcn-svelte components consistently
- Proper error handling with try-catch และ GraphQL error boundaries
- State management ด้วย Svelte stores
- Component composition with props และ snippets

**GraphQL:**

- Schema-first development approach
- Proper type definitions และ validation
- Use DataLoader pattern สำหรับ N+1 query prevention
- Implement proper authentication checks ใน resolvers

### Security Guidelines

- **Never commit secrets**: Use environment variables
- **Proper authentication**: Validate JWT tokens properly
- **Authorization**: Check user roles และ permissions ใน every resolver
- **Input validation**: Validate all inputs with Zod หรือ Go validation
- **SQL injection prevention**: Use GORM parameterized queries
- **XSS prevention**: Proper data sanitization ใน frontend

### Testing Strategy

**Backend Testing:**

```bash
# Unit tests สำหรับ services และ utilities
go test ./pkg/...

# Integration tests สำหรับ GraphQL resolvers
go test ./graph/...

# Database tests
go test ./internal/models/...
```

**Frontend Testing:**

```bash
# Component tests
npm run test

# E2E tests (if implemented)
npm run test:e2e
```

### Performance Considerations

- **Database**: Proper indexing, query optimization
- **GraphQL**: DataLoader สำหรับ batch loading
- **Frontend**: Lazy loading, component splitting
- **Caching**: Redis สำหรับ frequently accessed data
- **Images**: Proper optimization และ CDN usage

### General Guidelines

- **Language**: Always explain in Thai language (`อธิบายเป็นภาษาไทยเสมอ`)
- **Documentation**: Keep README และ CLAUDE.md updated
- **Version Control**: Use meaningful commit messages
- **Error Handling**: Proper error messages ใน both Thai และ English
- **Logging**: Structured logging สำหรับ debugging

## Environment Setup

### Required Environment Variables

**Backend (.env)**:

```env
DB_HOST=localhost                    # Database host
DB_PORT=5432                        # Database port
DB_USER=postgres                    # Database user
DB_PASSWORD=password                # Database password
DB_NAME=tru_activity               # Database name
REDIS_HOST=localhost               # Redis host
REDIS_PORT=6379                    # Redis port
JWT_SECRET=your-secret-key         # JWT signing key
JWT_EXPIRE_HOURS=24               # Token expiry
PORT=8080                         # Server port
ENV=development                   # Environment
CORS_ORIGINS=http://localhost:5173 # CORS origins
```

**Frontend (.env.local)**:

```env
VITE_API_URL=http://localhost:8080
VITE_GRAPHQL_URL=http://localhost:8080/query
```

## Troubleshooting

### Common Issues

1. **GraphQL Schema Changes**: Run `npm run codegen` ใน frontend เมื่อ schema เปลี่ยน
2. **Database Connection**: Check Docker containers are running
3. **CORS Issues**: Verify CORS_ORIGINS environment variable
4. **Authentication**: Check JWT_SECRET และ token expiry
5. **Hot Reload**: Use `air` สำหรับ backend hot reload

### Debug Commands

```bash
# Check service health
curl http://localhost:8080/health

# Check GraphQL endpoint
curl -X POST http://localhost:8080/query -H "Content-Type: application/json" -d '{"query":"query{__schema{types{name}}}"}'

# Database connection test
docker exec tru-activity-db pg_isready -U postgres

# Redis connection test
docker exec tru-activity-redis redis-cli ping
```

## Project Workflow Reminders

- **เมื่อทำการเขียนโค้ด หรือพัฒนาเสร็จให้ทำการ อัปเดตใน CLAUDE.md ทุกครั้ง**
- Run tests before committing
- Update documentation เมื่อ API changes
- Check security implications ของ new features
- Verify CORS และ authentication setup
- Test ทั้ง development และ production environments
