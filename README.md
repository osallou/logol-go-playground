# about

Logol rewriting in go

# status

in development

# deps

rabbitmq and redis

Needs cassiopee >= 1.0.9, org.irisa.genouest/cassiopee is built upon this version

# building binaries

    $ go build ./cmd/logolClient
    $ go build ./cmd/logolCassie
    $ go build ./cmd/logolMessage
    $ go build ./cmd/logolResult


# running

rabbitmq and redis connection url are given via env variables:

    LOGOL_REDIS_ADDR=localhost:6379
    LOGOL_RABBITMQ_ADDR=amqp://guest:guest@localhost:5672
    LOGOL_DEBUG=1 # to activate DEBUG log level, else level INFO


in cmd/logol[XXX]:

1 or more : logolMessage
1: logolResult
1: logolCassie

then logolClient

testmsg will send grammar and sequence info , once search is over it will stop processes
