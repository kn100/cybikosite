FROM golang:1.17.6 as builder

# Install some dependencies needed to build the project
# not sure why I need libgl-dev but whatever.

RUN mkdir /app
WORKDIR /app
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
ENV GO111MODULE=on
RUN go mod download

COPY . /app
RUN go build -o main .

WORKDIR /app
CMD ["/app/main"]