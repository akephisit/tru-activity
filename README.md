# TRU Activity - ระบบเก็บกิจกรรมมหาวิทยาลัย

ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี พัฒนาด้วย Go Fiber + GraphQL + PostgreSQL + SvelteKit

## ⚡ คุณสมบัติหลัก

### 🔐 Authentication & Authorization
- JWT-based authentication system
- Multi-level admin system (Super Admin, Faculty Admin, Regular Admin)
- Role-based access control และ permissions
- QR secret key สำหรับ users

### 🏛️ Organization Management
- จัดการคณะ (Faculties) และภาควิชา (Departments)
- User management พร้อม faculty/department assignments
- Admin role assignments และ permissions

### 📅 Activity Management
- สร้าง แก้ไข และจัดการกิจกรรม
- Activity types: Workshop, Seminar, Competition, Volunteer, Other
- Activity status tracking: Draft, Active, Completed, Cancelled
- Participation management พร้อม approval workflow
- Attendance tracking และ points system

### 📊 Dashboard & Analytics
- Role-based dashboard และ navigation
- Activity statistics และ participation metrics
- Real-time updates ด้วย GraphQL subscriptions
- Reports และ analytics สำหรับ admins

### 💳 Subscription System
- Subscription tracking พร้อม expiry management
- Multiple subscription types: Basic, Premium, VIP
- Status tracking: Active, Expired, Cancelled

## 🛠️ Tech Stack

### Backend
- **Go** - Programming language
- **Fiber v2** - High-performance web framework
- **GraphQL** - API query language (gqlgen)
- **PostgreSQL** - Primary database
- **Redis** - Caching และ session management
- **JWT** - Authentication tokens
- **GORM** - ORM สำหรับ database operations

### Frontend
- **SvelteKit** - Full-stack framework
- **Svelte 5** - Component framework พร้อม runes syntax
- **TypeScript** - Type-safe JavaScript
- **TailwindCSS** - Utility-first CSS framework
- **shadcn-svelte** - UI component library
- **Apollo Client** - GraphQL client
- **Zod** - Schema validation

### Infrastructure
- **Docker Compose** - Development environment
- **PostgreSQL** - Database container
- **Redis** - Cache container
- **Air** - Hot reloading สำหรับ Go development

## 🚀 การติดตั้งและรัน

### Prerequisites
- Docker และ Docker Compose
- Node.js 24.4.1+ (สำหรับ local frontend development)
- Go 1.24.5+ (สำหรับ local backend development)

### 1. Clone Repository
```bash
git clone https://github.com/kruakemaths/tru-activity.git
cd tru-activity
```

### 2. Setup Environment Variables
```bash
# Backend
cp backend/.env.example backend/.env

# Frontend (ถ้าต้องการ custom config)
# สร้าง .env.local ใน frontend/ directory
```

### 3. รันด้วย Docker Compose
```bash
# รัน development environment
docker-compose up -d

# ดู logs
docker-compose logs -f

# หยุด services
docker-compose down
```

### 4. เข้าใช้งานระบบ
- **Frontend**: http://localhost:5173
- **GraphQL Playground**: http://localhost:8080 (development only)
- **API Endpoint**: http://localhost:8080/query
- **Health Check**: http://localhost:8080/health

## 🏗️ โครงสร้างโปรเจค

```
/
├── backend/                 # Go Fiber + GraphQL Backend
│   ├── cmd/server/         # Application entry point
│   ├── internal/           # Internal packages
│   │   ├── config/        # Configuration management
│   │   ├── database/      # Database connection
│   │   ├── middleware/    # HTTP middleware
│   │   ├── models/        # Database models
│   │   └── handlers/      # HTTP handlers
│   ├── pkg/               # Public packages
│   │   ├── auth/          # JWT authentication
│   │   └── utils/         # Utility functions
│   ├── graph/             # GraphQL schema และ resolvers
│   ├── migrations/        # Database migrations
│   └── Dockerfile         # Backend container
├── frontend/              # SvelteKit Frontend
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/  # Svelte components
│   │   │   ├── graphql/     # GraphQL queries/mutations
│   │   │   └── stores/      # Svelte stores
│   │   └── routes/          # SvelteKit routes
│   └── Dockerfile.dev      # Frontend container
├── docker-compose.yml     # Development environment
└── README.md             # โปรเจค documentation
```

## 📊 Database Schema

### Core Tables
- **users** - ข้อมูลผู้ใช้ (students + admins)
- **faculties** - ข้อมูลคณะ
- **departments** - ข้อมูลภาควิชา
- **activities** - ข้อมูลกิจกรรม
- **participations** - การเข้าร่วมกิจกรรม
- **subscriptions** - ข้อมูล subscription (tracking expiry)

### Relationships
- Users belongsTo Faculty และ Department
- Activities belongsTo Faculty และ Department
- Participations เชื่อม Users กับ Activities
- Subscriptions belongsTo Users

## 🔧 Development

### Backend Development
```bash
cd backend

# Install dependencies
go mod tidy

# Run with hot reload
air

# Run tests
go test ./...

# Generate GraphQL code
go run github.com/99designs/gqlgen generate
```

### Frontend Development
```bash
cd frontend

# Install dependencies
npm install

# Generate GraphQL types
npm run codegen

# Run development server
npm run dev

# Type checking
npm run check

# Linting
npm run lint
```

### Database Operations
```bash
# Access PostgreSQL container
docker exec -it tru-activity-db psql -U postgres -d tru_activity

# Backup database
docker exec tru-activity-db pg_dump -U postgres tru_activity > backup.sql

# Restore database
docker exec -i tru-activity-db psql -U postgres tru_activity < backup.sql
```

## 👥 User Roles และ Permissions

### Student (นักศึกษา)
- ดูกิจกรรมที่เปิดรับสมัคร
- ลงทะเบียนเข้าร่วมกิจกรรม
- ดูประวัติการเข้าร่วมกิจกรรม
- ดูคะแนนและ subscription status

### Regular Admin (ผู้ดูแลทั่วไป)
- จัดการกิจกรรมในคณะ/ภาควิชาของตน
- อนุมัติการเข้าร่วมกิจกรรม
- บันทึกการเข้าร่วม (attendance)
- ดูรายงานกิจกรรม

### Faculty Admin (ผู้ดูแลคณะ)
- จัดการกิจกรรมทั้งคณะ
- จัดการผู้ใช้ในคณะ
- จัดการภาควิชาในคณะ
- ดูรายงานระดับคณะ

### Super Admin (ผู้ดูแลระบบ)
- จัดการทุกอย่างในระบบ
- จัดการคณะและภาควิชา
- จัดการผู้ใช้ทั้งหมด
- ดูรายงานทั้งระบบ

## 🔒 Security Features

- JWT token authentication พร้อม refresh mechanism
- Password hashing ด้วย bcrypt
- Role-based access control (RBAC)
- CORS protection
- SQL injection protection ด้วย GORM
- XSS protection ด้วย proper data sanitization

## 🚦 API Endpoints

### GraphQL Endpoint
- **URL**: `/query`
- **Method**: POST
- **Headers**: `Authorization: Bearer <token>`

### REST Endpoints
- **Health Check**: `GET /health`
- **GraphQL Playground**: `GET /` (development only)

## 📈 Monitoring และ Logging

### Health Checks
- Database connectivity
- Redis connectivity
- Service status

### Logging
- Structured logging ด้วย Go standard library
- Request/response logging
- Error tracking
- Performance metrics

## 🔧 Configuration

### Backend Environment Variables
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=tru_activity

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRE_HOURS=24

# Server
PORT=8080
ENV=development
CORS_ORIGINS=http://localhost:5173
```

### Frontend Environment Variables
```env
VITE_API_URL=http://localhost:8080
VITE_GRAPHQL_URL=http://localhost:8080/query
```

## 🤝 Contributing

1. Fork repository
2. สร้าง feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add some amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. เปิด Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

สำหรับคำถามหรือการสนับสนุน:
- GitHub Issues: [https://github.com/kruakemaths/tru-activity/issues](https://github.com/kruakemaths/tru-activity/issues)
- Email: support@example.com

---

พัฒนาโดย TRU Development Team 🚀