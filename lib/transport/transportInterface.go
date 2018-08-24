package logol

import (
    "log"
    "fmt"
    "os"
    "strconv"
    "sync"
    logol "org.irisa.genouest/logol/lib/types"
    logs "org.irisa.genouest/logol/lib/log"
)

var logger = logs.GetLogger("logol.transport")

const STEP_NONE int = -1
const STEP_PRE int = 0
const STEP_POST int = 1
const STEP_END int = 2
const STEP_BAN int = 3
const STEP_CASSIE int = 4
const STEP_YETTOBEDEFINED int = 5

type QueueType int
const QUEUE_MESSAGE = 0
const QUEUE_CASSIE = 1
const QUEUE_RESULT = 2
const QUEUE_EVENT = 3
const EXCHANGE_EVENT = 4

type MsgEvent struct {
    Step  int
}

const TRANSPORT_AMQP = 0
const TRANSPORT_ALLINONE = 1

var onceAnalyse sync.Once
var onceResult sync.Once
var onceCassie sync.Once
var tAnalyse Transport
var tResult Transport
var tCassie Transport

func getTransportKind() int {
    transportKind := TRANSPORT_AMQP
    osTransportKind := os.Getenv("LOGOL_TRANSPORT")
    if osTransportKind != "" {
        tk, err := strconv.Atoi(osTransportKind)
        if err == nil {
            transportKind = tk
        }
    }
    return transportKind
}



func GetTransport(kind QueueType) (Transport) {
    switch kind {
    case QUEUE_MESSAGE:
        onceAnalyse.Do(func(){
            transportKind := getTransportKind()
            if transportKind == TRANSPORT_AMQP {
                var t Transport
                t = NewTransportRabbit()
                tAnalyse = t
            }
        })
        return tAnalyse
    case QUEUE_RESULT:
        onceResult.Do(func(){
            transportKind := getTransportKind()
            if transportKind == TRANSPORT_AMQP {
                var t Transport
                t = NewTransportRabbit()
                tResult = t
            }
        })
        return tResult
    case QUEUE_CASSIE:
        onceCassie.Do(func(){
            transportKind := getTransportKind()
            if transportKind == TRANSPORT_AMQP {
                var t Transport
                t = NewTransportRabbit()
                tCassie = t
            }
        })
        return tCassie
    }
    return nil
}



type CallbackMessage func(data logol.Result) bool

func failOnError(err error, msg string) {
  if err != nil {
    log.Fatalf("%s: %s", msg, err)
    panic(fmt.Sprintf("%s: %s", msg, err))
  }
}

type Transport interface {
    Init(uid string)
    GetId() string
    GetProgress(uid string) (count int, ban int, match int)
    AddBan(uid string, nb int64)
    AddCount(uid string, nb int64)
    AddMatch(uid string, nb int64)
    SetBan(uid string, nb int64)
    SetCount(uid string, nb int64)
    SetMatch(uid string, nb int64)
    Clear(uid string)
    Close()
    PrepareMessage(logol.Result) string
    PublishMessage(queueName string, publish_msg string)
    PublishExchange(queueName string, publish_msg string)
    SendMessage(queue QueueType, data logol.Result) bool
    SendEvent(event MsgEvent) bool
    Listen(queueListen QueueType, fn CallbackMessage)
    GetGrammar(grammarId string) (g logol.Grammar, err bool)
    SetGrammar(grammarFile string, grammarId string) (err bool)
}
