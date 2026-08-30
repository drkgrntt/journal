# ---- build stage ----
FROM golang:1.24-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Pin templ to the exact version this module was built against
RUN go install github.com/a-h/templ/cmd/templ@v0.3.977
RUN templ generate
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/api

# ---- runtime stage ----
FROM alpine:3.20
# postgresql-client provides pg_dump, used by the nightly backup job.
# Alpine 3.20 ships pg_dump 16, newer than the host's Postgres 13 -- that's
# fine, pg_dump supports dumping from older server versions.
RUN apk add --no-cache ca-certificates tzdata postgresql-client \
    && adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/app ./app
USER appuser

ENTRYPOINT ["./app"]
