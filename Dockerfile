# Stage 1: Build React SPA
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build
# Output is in /build/assets

# Stage 2: Build Go binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22-alpine AS builder

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG APP_VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

WORKDIR /build

# Add Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Add sources
COPY . /build

# Copy built frontend assets
COPY --from=frontend-builder /build/assets ./assets/

# Build Go binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X 'main.appVersion=${APP_VERSION}' -X 'main.buildTime=${BUILD_TIME}' -X 'main.gitCommit=${GIT_COMMIT}'" -a -o wg-ui .

# Stage 3: Release
FROM alpine:3.20

RUN addgroup -S wgui && \
    adduser -S -D -G wgui wgui

RUN apk --no-cache add ca-certificates wireguard-tools iptables

WORKDIR /app

RUN mkdir -p db

# Copy binary files
COPY --from=builder --chown=wgui:wgui /build/wg-ui .
RUN chmod +x wg-ui
COPY init.sh .
RUN chmod +x init.sh

EXPOSE 5000/tcp
ENTRYPOINT ["./init.sh"]
