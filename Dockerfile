FROM golang:1.26-alpine

WORKDIR /app

EXPOSE 8080

CMD ["go", "run", "."]
