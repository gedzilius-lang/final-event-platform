# Multi-stage Next.js 14 production build for NiteOS Guest web app.
# Uses standalone output mode — self-contained node server.
# Builder uses Debian-based Node (glibc) so the SWC native binary works correctly.

FROM node:20-slim AS deps
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev

FROM node:20-slim AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
# Placeholder values for build — overridden at runtime
ENV SESSION_SECRET=build-placeholder-secret-at-least-32-chars
ENV GATEWAY_URL=http://localhost:8000
ENV AUTH_SERVICE_URL=http://localhost:8010
ENV NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_placeholder
ENV NEXT_PUBLIC_SITE_URL=http://localhost:3000
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
