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

## QR Code System และ Activity Management

ระบบ QR Code แบบ client-side generation และการจัดการกิจกรรมขั้นสูง:

### QR Code Features (Client-side Generation)

**QR Code System:**
- `backend/pkg/utils/qr.go` - Enhanced QR utilities พร้อม signature verification
- `backend/pkg/services/qr_service.go` - Complete QR validation และ scanning service
- Client-side QR generation (ไม่เก็บ QR images ใน server)
- เก็บแค่ unique identifier + secret ใน database
- QR data format: `{"student_id": "xxx", "timestamp": "xxx", "signature": "xxx", "version": 1}`

**Security Features:**
- HMAC-SHA256 signature validation
- Timestamp-based expiry (15 minutes)
- QR secret regeneration capability
- Server-side validation เมื่อสแกน QR
- Comprehensive scan logging

### Activity Management System

**Enhanced Activity Models:**
- `backend/internal/models/activity.go` - Extended activity models
- Activity templates สำหรับ reusable activities
- Recurring events พร้อม recurrence rules
- Activity assignments สำหรับ Regular Admins
- Faculty-scoped และ cross-faculty activities

**Activity Features:**
- Faculty-scoped activity creation
- Cross-faculty activities (Super Admin only)
- Activity templates และ recurring events
- Permission-based activity access
- Real-time participation tracking

### Activity Assignment System

**Assignment Management:**
- Faculty Admin assigns activities to Regular Admins
- Granular permissions (can_scan_qr, can_approve)
- Activity-specific access control
- Assignment tracking และ management
- Performance monitoring

### QR Scanning Flow

**Scanning Process:**
1. Admin scans QR code (มี timestamp และ signature)
2. Server validate signature และ check student exists
3. Record participation พร้อม attendance
4. Send real-time notification
5. Log scan attempt พร้อม metadata

**QR Scanner Interface:**
- `frontend/src/routes/dashboard/scanner/+page.svelte` - Mobile-friendly scanner
- Camera integration สำหรับ QR scanning
- Manual input fallback
- Real-time scan results
- Activity selection และ validation

### Student QR Code Interface

**My QR Code:**
- `frontend/src/routes/dashboard/my-qr/+page.svelte` - Student QR interface
- Auto-refreshing QR codes (15-minute expiry)
- QR secret regeneration
- Download และ copy functionality
- Security warnings และ instructions

### Real-time Notifications

**Notification System:**
- `backend/pkg/services/realtime_service.go` - WebSocket-based notifications
- Real-time QR scan notifications
- Participation updates
- Activity status changes
- Subscription-based filtering

**Notification Types:**
- QR scan results
- Participation approvals
- Activity assignments
- System announcements
- Health checks

### Activity Service

**Activity Management:**
- `backend/pkg/services/activity_service.go` - Advanced activity operations
- Template-based activity creation
- Recurring event generation
- Assignment management
- Faculty-scoped operations

**Recurring Events:**
- RRULE-based recurrence patterns
- Support for DAILY, WEEKLY, MONTHLY frequencies
- Weekday-specific recurrence
- Count และ until-date limits
- Parent-child activity relationships

### Database Enhancements

**New Tables:**
- `activity_templates` - Reusable activity templates
- `activity_assignments` - Admin assignments
- `qr_scan_logs` - Comprehensive scan logging
- Enhanced `activities` table พร้อม template support
- Enhanced `participations` table พร้อม QR scan data

**Migration:**
- `backend/migrations/003_qr_system_and_activity_enhancements.sql`

### GraphQL API Extensions

**New Operations:**
- QR data generation และ validation
- Activity template management
- Assignment operations
- Real-time subscriptions
- Scan logging queries

**Permission Integration:**
- Role-based QR scanning permissions
- Faculty-scoped activity access
- Assignment-based permissions
- Real-time notification filtering

### Mobile Optimization

**Mobile Features:**
- Touch-friendly QR scanner interface
- Camera integration พร้อม fallbacks
- Offline QR code storage
- Responsive design สำหรับ mobile devices
- Progressive Web App capabilities

### Security Implementation

**QR Security:**
- Client-side generation ลด server load
- Cryptographic signatures ป้องกัน tampering
- Time-based expiry ลด replay attacks
- Comprehensive audit logging
- Secret rotation capability

### Development Workflow

- เมื่อทำการเขียนโค้ด หรือพัฒนาเสร็จให้ทำการ อัปเดตใน CLAUDE.md ทุกครั้ง