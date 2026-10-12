# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Copy module files first so the dependency layer is cached between builds.
COPY go.mod go.sum* ./
COPY . .
# `go mod tidy` makes the build work even if go.sum was not committed;
# commit go.sum for reproducible builds.
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/server ./cmd/mockhis

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
USER app
# mockhis is only started by docker-compose for development (see the mock-his service).
COPY --from=build /out/server /out/mockhis /usr/local/bin/

EXPOSE 8080
ENTRYPOINT ["server"]
