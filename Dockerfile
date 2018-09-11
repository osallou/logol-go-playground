FROM debian:sid
RUN apt-get update && apt-get install -y golang git libcassie-dev swig libboost-dev
RUN mkdir -p /tmp/src/github.com/osallou/logol-go-playground/cmd /tmp/src/github.com/osallou/logol-go-playground/lib
ENV GOPATH=/tmp
WORKDIR /tmp/src/github.com/osallou/logol-go-playground
COPY cmd/ ./cmd/
COPY lib/ ./lib/
RUN go get ./...

RUN GOOS=linux go build -a ./cmd/logolClient/logolClient.go
RUN GOOS=linux go build -a ./cmd/logolCassie/logolCassie.go
RUN GOOS=linux go build -a ./cmd/logolMessage/logolMessage.go
RUN GOOS=linux go build -a ./cmd/logolResult/logolResult.go

FROM debian:sid
RUN apt-get update && apt-get install -y libcassie1v5
WORKDIR /usr/bin/
COPY --from=0 /tmp/src/github.com/osallou/logol-go-playground/logolClient .
COPY --from=0 /tmp/src/github.com/osallou/logol-go-playground/logolCassie .
COPY --from=0 /tmp/src/github.com/osallou/logol-go-playground/logolMessage .
COPY --from=0 /tmp/src/github.com/osallou/logol-go-playground/logolResult .
ENV LOGOL_REDIS_ADDR=localhost:6379
ENV LOGOL_RABBITMQ_ADDR=amqp://guest:guest@localhost:5672
ENV LOGOL_DEBUG=0
