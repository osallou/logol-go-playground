FROM debian:sid
RUN apt-get update && apt-get install -y golang git libcassie-dev swig libboost-dev
RUN mkdir -p /tmp/src/org.irisa.genouest/logol/cmd /tmp/src/org.irisa.genouest/logol/lib
ENV GOPATH=/tmp
WORKDIR /tmp/src/org.irisa.genouest/logol
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
COPY --from=0 /tmp/src/org.irisa.genouest/logol/logolClient .
COPY --from=0 /tmp/src/org.irisa.genouest/logol/logolCassie .
COPY --from=0 /tmp/src/org.irisa.genouest/logol/logolMessage .
COPY --from=0 /tmp/src/org.irisa.genouest/logol/logolResult .
ENV LOGOL_REDIS_ADDR=localhost:6379
ENV LOGOL_RABBITMQ_ADDR=amqp://guest:guest@localhost:5672
ENV LOGOL_DEBUG=0
