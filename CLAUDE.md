# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Structure

This is a SvelteKit frontend application located in the `frontend/` directory. The main application code is in `frontend/src/`.

### Key Directories

- `frontend/src/routes/` - SvelteKit routes and pages
- `frontend/src/lib/components/` - Reusable Svelte components including UI library
- `frontend/src/lib/components/ui/` - shadcn-svelte UI component library

## Common Development Commands

All commands should be run from the `frontend/` directory:

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
```

## Architecture Overview

### UI Framework Stack

- **SvelteKit**: Full-stack framework with file-based routing
- **Svelte 5**: Component framework with runes syntax
- **TailwindCSS**: Utility-first CSS framework
- **shadcn-svelte**: UI component library (bits-ui based)

### Data Management

- **Zod**: Schema validation (schemas defined in `lib/components/schemas.ts`)
- **TanStack Table**: Data table functionality with sorting, filtering, selection
- **Mock Data**: Static data arrays (e.g., `routes/dashboard-01/data.ts`)

### Key Components

- **DataTable**: Advanced table with drag-and-drop, selection, and custom cell renderers
- **Sidebar**: Navigation with collapsible sidebar using bits-ui
- **Charts**: Interactive charts using LayerChart and D3
- **UI Components**: Complete shadcn-svelte component library

### Routing Structure

- Root page (`/`) displays basic SvelteKit welcome
- Dashboard (`/dashboard-01`) contains the main application with sidebar, data table, and charts

### Development Patterns

- Components use Svelte 5 runes syntax (`$props()`, `$state()`)
- TypeScript throughout with proper type definitions
- Tailwind classes for styling with utility-first approach
- Component composition with props and snippets for flexibility

## Data Schema

The main data model is defined in `lib/components/schemas.ts` with fields:

- `id`: number (unique identifier)
- `header`: string (item title)
- `type`: string (content type category)
- `status`: string (workflow status)
- `target`: string (target value)
- `limit`: string (limit value)
- `reviewer`: string (assigned reviewer name)

## Documentation and Version Checking use context7

### Library Documentation Lookup

- **SvelteKit**: `/sveltejs/kit` - Official documentation for SvelteKit framework
- **Svelte**: `/sveltejs/svelte` - Core Svelte documentation
- **TailwindCSS**: `tailwindcss.com/docs` - Utility-first CSS framework docs
- **Zod**: `/colinhacks/zod` - Schema validation library documentation
- **shadcn-svelte**: `/https://www.shadcn-svelte.com/docs` - UI component library documentation

## Development Guidelines

- Always explain in Thai language
  - `อธิบายเป็นภาษาไทยเสมอ`

## Project Workflow Reminders

- เมื่อทำการเขียนโค้ด หรือพัฒนาเสร็จให้ทำการ อัปเดตใน CLAUDE.md ทุกครั้ง