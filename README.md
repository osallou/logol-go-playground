# about

Logol rewriting in go.

Logol is a biological pattern matching tool against DNA or protein sequences.
It matches a grammar againts one or several sequences to find patterns (exact match or with allowed morphisms, substitutions or indels).

Original Logol was written in Prolog and hosted at https://github.com/genouest/logol.

## status

in development

## deps

rabbitmq and redis

Needs cassiopee >= 1.0.9, github.com/osallou/cassiopee-go is built upon this version

## building binaries

    $ go build ./cmd/logolClient
    $ go build ./cmd/logolCassie
    $ go build ./cmd/logolMessage
    $ go build ./cmd/logolResult

### setting version

    go build -ldflags "-X github.com/osallou/logol-go-playground/lib/utils.Buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X github.com/osallou/logol-go-playground/lib/utils.Githash=`git rev-parse HEAD` -X github.com/osallou/logol-go-playground/lib/utils.Version=0.1" ./cmd/logolClient

## running

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

## stats

to enable stats , LOGOL_STATS env variable should be set on processes.

a *dot* file will be generated to represent the graph of search

*logolPrometheus* can be optionally started (won't be stopped automatically) to record match time statistics and expose its metrics to prometheus. Option --listen specifies to port to listen to, then metrics are accessible via http://localhost:port/metrics

To enable prometheus statistics during logol search, one must set env variable LOGOL_PROM to the url of *logolPrometheus*, example:

    LOGOL_PROM=http://localhost:8080 logolMessage ...

If not sent, no statistic will be sent.

**Warning**: Enabling some statistics will impact performance and should be activated only for debug/analysis.


## show workflow

To get visual workflow, you can run in *fake* mode

    LOGOL_STATS=1 LOGOL_FAKE=1 go run cmd/logolClient/logolClient.go -grammar testdata/grammar.txt -sequence sequence.txt -standalone 1
    dot logol-ed31188b-0a22-4b1b-b76a-86481df615a4.stats.dot -Tpng -o fake.png

## doc

godoc can be accessed locally:

    godoc -http=:6060
    => http://localhost:6060/pkg/github.com/osallou/logol-go-playground

## testing

    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html