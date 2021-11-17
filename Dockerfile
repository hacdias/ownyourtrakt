FROM golang:1.17-alpine3.14 as build
RUN apk update && \
    apk add --no-cache git gcc g++ musl-dev
WORKDIR /ownyourtrakt/
COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download
COPY . /ownyourtrakt/
RUN go build -o main

FROM alpine:3.12
COPY --from=build /ownyourtrakt/main /bin/ownyourtrakt
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /app
EXPOSE 8050
CMD ["ownyourtrakt"]
