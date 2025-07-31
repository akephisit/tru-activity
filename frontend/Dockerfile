# Cloud Run Dockerfile for SvelteKit Frontend
FROM node:24.4.1-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies for build
RUN npm ci && npm cache clean --force

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage - Use Node.js server for Cloud Run
FROM node:24.4.1-alpine AS production

# Install dependencies needed for production
RUN apk add --no-cache curl

# Create non-root user
RUN addgroup -g 1001 -S svelte && \
    adduser -S svelte -u 1001

# Set working directory
WORKDIR /app

# Copy built application and dependencies
COPY --from=builder /app/build build/
COPY --from=builder /app/node_modules node_modules/
COPY --from=builder /app/package.json .

# Change ownership to non-root user
RUN chown -R svelte:svelte /app
USER svelte

# Expose port 3000 (SvelteKit default)
EXPOSE 3000

# Health check for Cloud Run
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Start the Node.js server
CMD ["node", "build"]