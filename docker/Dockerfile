FROM node:22-alpine AS ui

WORKDIR /src/ui

# Copy package.json and lock file then install dependencies
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile

# COPY the ui and build
COPY ui/ ./
RUN pnpm run build

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM golang:1.25-alpine AS backend

ARG VERSION
ARG COMMIT

RUN apk add --no-cache \
    gcc \
    git \
    libwebp-dev \
    musl-dev \
    sqlite-dev

ENV CGO_ENABLED=1 

WORKDIR /src/

COPY . .
RUN rm -rf ui

COPY --from=ui /src/ui/build ./ui/build
COPY --from=ui /src/ui/embed.go ./ui/embed.go

RUN go mod download
RUN set -eux; \
    SET_VERSION="${VERSION:-$(git describe --tags --exact-match 2>/dev/null || echo dev)}"; \
    SET_COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"; \
    echo "Building with VERSION=${SET_VERSION}, COMMIT=${SET_COMMIT}"; \
    go build -ldflags "-X github.com/geerew/off-course/version.Version=${SET_VERSION} -X github.com/geerew/off-course/version.Commit=${SET_COMMIT}" -o offcourse .


# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    libwebp \
    sqlite-libs \
    libva-utils \
    mesa-va-gallium

COPY  --from=backend /src/offcourse /usr/local/bin/offcourse

RUN chmod +x /usr/local/bin/offcourse

EXPOSE 80

VOLUME [ "/offcourse" ]

ENTRYPOINT [ "/usr/local/bin/offcourse", "serve", "--data-dir", "/offcourse", "--http", "0.0.0.0:80" ]