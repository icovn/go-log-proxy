# 1️⃣ Use official Golang image as the build stage
FROM golang:1.23 AS builder

# 2️⃣ Set the working directory inside the container
WORKDIR /app

# 3️⃣ Copy the Go source files and dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY config.go main.go ./

# 4️⃣ Build the Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# 5️⃣ Use a lightweight image for the final container
FROM alpine:latest

RUN apk add libc6-compat

# 6️⃣ Set working directory inside the new container
WORKDIR /app

# 7️⃣ Copy the built binary from the builder stage
COPY --from=builder /app/main .

# 8️⃣ Expose the application port
EXPOSE 8080
EXPOSE 9000

# 9️⃣ Run the Go application
CMD ["./main"]
