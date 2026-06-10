# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -a -installsuffix cgo \
  -ldflags "-s -w -extldflags -static -X main.commit=${GIT_COMMIT}" \
  -o /out/batch .

FROM alpine:3.23

RUN addgroup -S app && adduser -S -G app -u 10001 app \
  && apk add --no-cache tini ca-certificates tzdata \
  && cp /usr/share/zoneinfo/Asia/Bangkok /etc/localtime \
  && echo "Asia/Bangkok" > /etc/timezone \
  && update-ca-certificates

WORKDIR /home/app

COPY --from=build /out/batch /home/app/batch

USER app

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/home/app/batch"]
