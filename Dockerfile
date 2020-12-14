FROM golang:latest As builder
LABEL maintainer="James Tsang"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main .

FROM golang:latest As final

EXPOSE 8080
USER 65534

COPY --from=builder /app/main main

CMD ["./main"]