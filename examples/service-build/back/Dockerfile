FROM golang:alpine AS build
ARG MESSAGE
WORKDIR /go/src
COPY main.go .
RUN CGO_ENABLED=0 go build -o /server -ldflags="-X main.Message=${MESSAGE} -s -w" main.go

FROM scratch AS run
ENTRYPOINT ["/server"]
COPY --from=build /server /server
