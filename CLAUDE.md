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

## Development References

### Context and Library Guidelines

- ถ้าจะเพิ่ม library ใหม่ หรือ ตรวจสอบ document ให้ดูจาก context7
- ก่อนที่จะเริ่มเขียนโค้ด ให้ทำการตรวจสอบ document วิธีการเขียนทุกครั้งโดยใช้ context7

### Development Workflow

- เมื่อทำการเขียนโค้ด หรือพัฒนาเสร็จให้ทำการ อัปเดตใน CLAUDE.md ทุกครั้ง