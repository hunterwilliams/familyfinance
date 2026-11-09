FROM node:22-alpine AS builder

ARG VITE_API_KEY=dev-api-key
ENV VITE_API_KEY=$VITE_API_KEY

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.27-alpine

RUN apk add --no-cache wget

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

# Use 127.0.0.1 — nginx listens on IPv4 only, and localhost can resolve to ::1.
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:80/ || exit 1
