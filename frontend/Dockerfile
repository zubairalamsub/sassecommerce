# syntax=docker/dockerfile:1.7
FROM node:20-alpine AS base

# Install dependencies
FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm npm ci

# Build
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# Service URLs are baked into Next.js rewrites at build time.
# Defaults match docker-compose; override via --build-arg for other targets.
ARG TENANT_SERVICE_URL=http://tenant-service:8080
ARG USER_SERVICE_URL=http://user-service:8082
ARG PRODUCT_SERVICE_URL=http://product-service:8083
ARG INVENTORY_SERVICE_URL=http://inventory-service:8084
ARG PAYMENT_SERVICE_URL=http://payment-service:8085
ARG SHIPPING_SERVICE_URL=http://shipping-service:8086
ARG NOTIFICATION_SERVICE_URL=http://notification-service:8087
ARG REVIEW_SERVICE_URL=http://review-service:8088
ARG CART_SERVICE_URL=http://cart-service:8089
ARG SEARCH_SERVICE_URL=http://search-service:8090
ARG PROMOTION_SERVICE_URL=http://promotion-service:8091
ARG VENDOR_SERVICE_URL=http://vendor-service:8092
ARG ANALYTICS_SERVICE_URL=http://analytics-service:8093
ARG RECOMMENDATION_SERVICE_URL=http://recommendation-service:8094
ARG CONFIG_SERVICE_URL=http://config-service:8095
ARG ORDER_SERVICE_URL=http://order-service:8080
ARG PROMETHEUS_SERVICE_URL=http://ecommerce-prometheus:9090
ENV TENANT_SERVICE_URL=$TENANT_SERVICE_URL \
    USER_SERVICE_URL=$USER_SERVICE_URL \
    PRODUCT_SERVICE_URL=$PRODUCT_SERVICE_URL \
    INVENTORY_SERVICE_URL=$INVENTORY_SERVICE_URL \
    PAYMENT_SERVICE_URL=$PAYMENT_SERVICE_URL \
    SHIPPING_SERVICE_URL=$SHIPPING_SERVICE_URL \
    NOTIFICATION_SERVICE_URL=$NOTIFICATION_SERVICE_URL \
    REVIEW_SERVICE_URL=$REVIEW_SERVICE_URL \
    CART_SERVICE_URL=$CART_SERVICE_URL \
    SEARCH_SERVICE_URL=$SEARCH_SERVICE_URL \
    PROMOTION_SERVICE_URL=$PROMOTION_SERVICE_URL \
    VENDOR_SERVICE_URL=$VENDOR_SERVICE_URL \
    ANALYTICS_SERVICE_URL=$ANALYTICS_SERVICE_URL \
    RECOMMENDATION_SERVICE_URL=$RECOMMENDATION_SERVICE_URL \
    CONFIG_SERVICE_URL=$CONFIG_SERVICE_URL \
    ORDER_SERVICE_URL=$ORDER_SERVICE_URL \
    PROMETHEUS_SERVICE_URL=$PROMETHEUS_SERVICE_URL
RUN npm run build

# Production
FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Media storage directory — mount a named Docker volume here
RUN mkdir -p /app/media && chown nextjs:nodejs /app/media
ENV MEDIA_STORAGE_PATH=/app/media
ENV NEXT_PUBLIC_MEDIA_URL=/api/media

USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
