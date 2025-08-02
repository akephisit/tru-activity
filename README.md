# TRU Activity

ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี

Full-stack web application ที่พัฒนาด้วย Go Fiber + GraphQL + PostgreSQL + SvelteKit สำหรับจัดการกิจกรรมมหาวิทยาลัย พร้อม multi-level admin system และ role-based access control

## 🚀 Quick Start - Development

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- Git

### 1. Clone Repository
```bash
git clone <repository-url>
cd tru-activity
```

### 2. Start Development Environment
```bash
# Start all services (recommended)
./scripts/dev.sh start

# Or just start backend services
./scripts/dev.sh backend

# Or start frontend only (requires backend running)
./scripts/dev.sh frontend
```

### 3. Access Applications
- **Frontend (Vite Dev Server)**: http://localhost:5173
- **Backend API**: http://localhost:8080
- **GraphQL Playground**: http://localhost:8080/
- **Database**: localhost:5432 (postgres/devpassword123)
- **Redis**: localhost:6379

### 4. Development Commands
```bash
# View logs
./scripts/dev.sh logs

# Stop all services
./scripts/dev.sh stop

# Restart services
./scripts/dev.sh restart

# Check service status
./scripts/dev.sh status

# Reset database (deletes all data)
./scripts/dev.sh reset-db
```

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