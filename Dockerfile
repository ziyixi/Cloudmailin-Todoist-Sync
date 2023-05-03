############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk add git
WORKDIR /go/src/app
COPY . .
# Build the binary.
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o cloudmailin_todoist_sync_app ./src
############################
# STEP 2 build a small image
############################
FROM scratch
LABEL org.opencontainers.image.authors="docker@ziyixi.science"
LABEL org.opencontainers.image.source=https://github.com/ziyixi/Cloudmailin-Todoist-Sync
LABEL org.opencontainers.image.description="Package to handle cloudmailin's post on receiving emails, and update todoist tasks accordingly with ChatGPT's summary"
LABEL org.opencontainers.image.licenses=MIT

ENV PORT=8080
# Copy our static executable.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/app/cloudmailin_todoist_sync_app /cloudmailin_todoist_sync_app

# Run the app binary.
EXPOSE 8080
ENTRYPOINT ["/cloudmailin_todoist_sync_app"]