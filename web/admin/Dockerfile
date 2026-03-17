# Multi-stage Next.js 14 production build for NiteOS Admin console.
# Uses standalone output mode — self-contained node server, no separate next start needed.
# Builder uses Debian-based Node (glibc) so the SWC native binary works correctly.

FROM node:20-slim AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

FROM node:20-slim AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
# Placeholder values required for build-time type-check only; overridden at runtime.
ENV SESSION_SECRET=build-placeholder-secret-at-least-32-chars
ENV GATEWAY_URL=http://localhost:8000
ENV AUTH_SERVICE_URL=http://localhost:8010
ENV ADMIN_ORIGIN=http://localhost:3001
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

# standalone output includes only what is needed to run the server.
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3001
ENV PORT=3001
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
