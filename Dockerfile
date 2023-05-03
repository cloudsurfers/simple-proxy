FROM golang:alpine AS builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

WORKDIR /build
# Copy and download dependency using go mod
COPY ./src/* ./
COPY ./go.mod ./
#RUN go mod download

COPY ./src .

# Build the application
RUN go build -a -tags musl -ldflags="-extldflags=-static" -o main .

######################################

FROM scratch
COPY --from=builder /build/main /app/simple-proxy
#HEALTHCHECK --interval=15s --timeout=15s  CMD [ "./simple-proxy", "-h=http://localhost:3000/health" ]

WORKDIR /app
EXPOSE 3000
CMD ["./simple-proxy"]
