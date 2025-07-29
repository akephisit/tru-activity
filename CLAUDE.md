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

## Authentication และ Authorization System

### Permission System Implementation

ระบบ Authentication และ Authorization ที่รองรับ 3 ระดับ admin:

**1. Super Admin** - จัดการระบบทั้งหมด:
- CRUD คณะทั้งหมด
- จัดการ subscription status (tracking เท่านั้น)
- ดูรายงานรวมระบบ
- เพิ่ม/ลบ Faculty Admins

**2. Faculty Admin** - จัดการคณะตัวเอง:
- จัดการนักศึกษาและกิจกรรมในคณะ
- เพิ่ม/ลบ Regular Admins
- ดูรายงานคณะ
- จัดการ departments
- รับแจ้งเตือน subscription expiry

**3. Regular Admin** - ดำเนินการระดับกิจกรรม:
- สแกน QR codes
- ดูข้อมูลกิจกรรมที่ได้รับมอบหมาย

### Key Implementation Files

**Backend Permission System:**
- `backend/pkg/permissions/permissions.go` - Permission system หลัก
- `backend/internal/middleware/graphql_auth.go` - GraphQL auth middleware
- `backend/pkg/auth/jwt.go` - JWT service (ขยายด้วย faculty_id, department_id)
- `backend/internal/models/user.go` - User model พร้อม permission methods

**GraphQL Integration:**
- `backend/graph/schema.graphqls` - Schema พร้อม auth directives (@auth, @hasRole)
- `backend/graph/schema.resolvers.go` - Resolvers พร้อม authorization
- `backend/graph/model/models_gen.go` - Generated GraphQL models
- `backend/graph/generated/generated.go` - Generated GraphQL interfaces

### Permission Features

**Role-based Access:**
- Field-level authorization ใน GraphQL
- Faculty-scoped data access
- Flexible permission checking
- Role hierarchy validation

**Security Features:**
- JWT tokens พร้อม faculty และ department info
- Context-based authorization
- Input validation และ sanitization
- Proper error handling

### Usage Examples

```go
// ตรวจสอบ permission
authCtx, err := middleware.RequirePermission(ctx, permissions.PermCreateActivity)

// ตรวจสอบ role
authCtx, err := middleware.RequireRole(ctx, models.UserRoleFacultyAdmin)

// ตรวจสอบ faculty permission
authCtx, err := middleware.RequireFacultyPermission(ctx, permissions.PermCreateActivity, facultyID)
```

## Development References

### Context and Library Guidelines

- ถ้าจะเพิ่ม library ใหม่ หรือ ตรวจสอบ document ให้ดูจาก context7
- ก่อนที่จะเริ่มเขียนโค้ด ให้ทำการตรวจสอบ document วิธีการเขียนทุกครั้งโดยใช้ context7

## Faculty Management และ Subscription System

ระบบจัดการคณะและการแจ้งเตือน subscription ที่ครอบคลุม:

### Faculty Management Features

**Backend Implementation:**
- `backend/internal/models/faculty.go` - Faculty และ Department models
- `backend/internal/models/subscription.go` - Enhanced subscription model พร้อม notification tracking
- `backend/internal/models/analytics.go` - Analytics และ metrics models
- `backend/graph/schema.graphqls` - Extended GraphQL schema
- `backend/graph/schema.resolvers.go` - Complete CRUD resolvers

**Key Features:**
- CRUD operations สำหรับ faculties และ departments
- Faculty admin assignment และ role management
- Department management with faculty association
- Real-time data tracking และ analytics

### Subscription Notification System

**Notification Service:**
- `backend/pkg/notifications/notification_service.go` - Complete notification service
- Email notifications สำหรับ subscription expiry (7 วัน, 1 วัน)
- Automatic notification scheduling
- Notification logging และ status tracking

**Subscription Plans:**
- Basic, Premium, Enterprise plans (display only)
- Faculty-based subscriptions (แทน user-based)
- Expiry tracking พร้อม automatic notifications
- Non-blocking system - ไม่ limit การใช้งาน

### Analytics และ Metrics

**System Metrics:**
- Total faculties, departments, students
- Activity และ participation statistics
- Subscription status overview
- Daily metrics collection

**Faculty Metrics:**
- Student และ activity counts per faculty
- Attendance averages
- Performance tracking
- Historical data storage

### Frontend Components

**Super Admin Dashboard:**
- `frontend/src/routes/dashboard/admin/+page.svelte` - System overview
- Real-time metrics display
- Subscription expiry alerts
- Faculty performance overview

**Faculty Management:**
- `frontend/src/routes/dashboard/faculties/+page.svelte` - Faculty CRUD
- Department management
- Admin assignment interface
- Performance metrics display

**Subscription Management:**
- `frontend/src/routes/dashboard/subscriptions/+page.svelte` - Subscription overview
- Expiry notifications
- Plan management (tracking only)
- Status monitoring

### Database Schema

**New Tables:**
- `faculty_metrics` - Faculty performance data
- `system_metrics` - System-wide analytics
- `notification_logs` - Notification history
- Enhanced `subscriptions` table พร้อม notification fields

**Migration:**
- `backend/migrations/002_update_subscriptions_and_analytics.sql`

### Permission Integration

ระบบใช้ existing permission system:
- Super Admin: Full access ทุก features
- Faculty Admin: Faculty-scoped access
- Regular Admin: Limited operation access

### Development Workflow

- เมื่อทำการเขียนโค้ด หรือพัฒนาเสร็จให้ทำการ อัปเดตใน CLAUDE.md ทุกครั้ง