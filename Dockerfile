FROM golang:1.22 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -C cmd -o /bin/app

FROM ubuntu:latest
RUN apt-get update
RUN apt-get install -y ca-certificates
RUN update-ca-certificates

COPY --from=build /bin/app /bin/app

EXPOSE 5000
CMD ["/bin/app"]