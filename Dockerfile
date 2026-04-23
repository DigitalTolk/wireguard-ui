# Stage 1: Build React SPA
FROM node:25-alpine AS frontend
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY src/ ./src/
COPY public/ ./public/
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts components.json ./
RUN npm run build
# Output: /src/assets/

# Stage 2: Build Go binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26-alpine AS backend

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG APP_VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

WORKDIR /src

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source (assets excluded via .dockerignore)
COPY . .

# Inject frontend build output for go:embed
COPY --from=frontend /src/assets ./assets/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X 'main.appVersion=${APP_VERSION}' -X 'main.buildTime=${BUILD_TIME}' -X 'main.gitCommit=${GIT_COMMIT}'" \
    -a -o wg-ui .

# Stage 3: Runtime
FROM alpine:3.23

RUN addgroup -S wgui && adduser -S -D -G wgui wgui
RUN apk --no-cache add ca-certificates wireguard-tools iptables

WORKDIR /app
RUN mkdir -p db

COPY --from=backend --chown=wgui:wgui /src/wg-ui .
COPY --chown=wgui:wgui init.sh .
RUN chmod +x wg-ui init.sh

USER wgui
EXPOSE 5000/tcp
ENTRYPOINT ["./init.sh"]
