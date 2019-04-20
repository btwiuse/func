FROM golang:1.12.4 AS build-env
WORKDIR /src
ARG GOLANGCI_LINT_TAG=v1.16.0
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -d -b $GOPATH/bin $GOLANGCI_LINT_TAG
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test -cover -race ./...
RUN golangci-lint run -v ./...
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/func

FROM scratch
WORKDIR /app
COPY --from=build-env /out/func .
ENTRYPOINT ["/app/func"]
