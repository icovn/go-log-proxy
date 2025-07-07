# go-log-proxy
A middleware component that receives logs from HTTP, TCP, and then sends them to Console, File, Loki, and so much more.

## Init
```shell
go mod init simple_log_proxy
```

## Dependencies

```shell
## Environment variables
go get github.com/joho/godotenv
```

## Hot reload

```shell
## Install air via go install
go install github.com/air-verse/air@latest

## Generate configuration file
air init

## Run the Gin Server with air
air
```

## Build

```shell
## Build
docker build --platform linux/amd64 -t simple-log-proxy .

## Run
docker run --rm -p 8080:8080 -p 9090:9090 --env-file .env simple-log-proxy

## Build and push
docker build --platform linux/amd64 -t icovn/simple_log_proxy:1.0.0 .
docker push icovn/simple_log_proxy:1.0.0
```