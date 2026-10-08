FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.26-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH=arm64
ARG WARPSCOUT_REPO=niklzz/warpscout-tg
ARG WARPSCOUT_REF=master
RUN apk add --no-cache git ca-certificates
WORKDIR /src
RUN case "$WARPSCOUT_REPO" in niklzz/warpscout-tg|vernette/warpscout) ;; *) echo "Unsupported WARPSCOUT_REPO" >&2; exit 1 ;; esac \
    && git init . && git remote add origin "https://github.com/$WARPSCOUT_REPO.git" \
    && git fetch --depth=1 origin "$WARPSCOUT_REF" && git checkout --detach FETCH_HEAD \
    && git rev-parse HEAD > /source-commit.txt
RUN GODEBUG=http2client=0 go mod download -x
RUN CGO_ENABLED=0 GOMAXPROCS=2 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -p 1 -trimpath -ldflags="-s -w" -o /warpscout .

COPY runner.go runner_test.go /runner/
RUN cd /runner && GO111MODULE=off go test -v -p 1 .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /runner-bin /runner/runner.go
RUN mkdir -p /runtime-dirs/output /runtime-dirs/state /runtime-dirs/tmp && chmod 1777 /runtime-dirs/tmp

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /warpscout /usr/local/bin/warpscout
COPY --from=build /source-commit.txt /app/source-commit.txt
COPY --from=build /runner-bin /app/runner
COPY --from=build /runtime-dirs/output /output
COPY --from=build /runtime-dirs/state /state
COPY --from=build /runtime-dirs/tmp /tmp
WORKDIR /state
ENTRYPOINT ["/app/runner"]
