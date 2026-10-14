FROM golang:1.26-alpine AS dev

WORKDIR /app

EXPOSE 8080

CMD ["go", "run", "."]

# Compile on the builder's own architecture and cross-compile for the target
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /out/api .

# Final stage, used by default. Build the dev image with: --target dev
FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app && mkdir /data && chown app:app /data

COPY --from=build /out/api /usr/local/bin/api

USER app
ENV DB_PATH=/data/app.db
VOLUME /data
EXPOSE 8080

CMD ["api"]