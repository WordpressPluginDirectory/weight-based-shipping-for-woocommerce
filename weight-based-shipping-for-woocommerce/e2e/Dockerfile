FROM golang:alpine3.19 as builder

WORKDIR /tmp/src

COPY go.mod go.sum ./

RUN PWGO_VER=$(grep -oE "playwright-go v\S+" go.mod | sed 's/playwright-go //g') \
    && go install github.com/playwright-community/playwright-go/cmd/playwright@${PWGO_VER}

RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    go test -c -o /test ./src


FROM ubuntu:jammy

COPY --from=builder /go/bin/playwright  /
RUN <<EOF
    set -e
    apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
    /playwright install --with-deps chromium
EOF

COPY --from=builder /test /
CMD ["/test", "-test.v"]