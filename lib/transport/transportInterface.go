package logol

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	logs "github.com/osallou/logol-go-playground/lib/log"
	logol "github.com/osallou/logol-go-playground/lib/types"
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
const QUEUE_LOG = 5

type MsgEvent struct {
	Step int
}

const TRANSPORT_AMQP = 0
const TRANSPORT_ALLINONE = 1

var onceAnalyse sync.Once
var onceResult sync.Once
var onceCassie sync.Once
var onceLog sync.Once
var tAnalyse Transport
var tResult Transport
var tCassie Transport
var tLog Transport

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

func GetTransport(kind QueueType) Transport {
	switch kind {
	case QUEUE_MESSAGE:
		onceAnalyse.Do(func() {
			transportKind := getTransportKind()
			if transportKind == TRANSPORT_AMQP {
				var t Transport
				t = NewTransportRabbit()
				tAnalyse = t
			}
		})
		return tAnalyse
	case QUEUE_RESULT:
		onceResult.Do(func() {
			transportKind := getTransportKind()
			if transportKind == TRANSPORT_AMQP {
				var t Transport
				t = NewTransportRabbit()
				tResult = t
			}
		})
		return tResult
	case QUEUE_CASSIE:
		onceCassie.Do(func() {
			transportKind := getTransportKind()
			if transportKind == TRANSPORT_AMQP {
				var t Transport
				t = NewTransportRabbit()
				tCassie = t
			}
		})
		return tCassie
	case QUEUE_LOG:
		onceLog.Do(func() {
			transportKind := getTransportKind()
			if transportKind == TRANSPORT_AMQP {
				var t Transport
				t = NewTransportRabbit()
				tLog = t
			}
		})
		return tLog
	}
	return nil
}

type CallbackMessage func(data logol.Result) bool
type CallbackLog func(data string) bool

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

// Transport is the interface that wraps message between search processes
//
// Init initialize the transport
// GetId returns the identifier of current transport
// GetQueueStatus returns the number of pending messages and number of consumers
// GetProgress returns the number of final match, rejected matches and number of possibilities
// Clear cleans temporary data
// Close closes the transport
// PrepareMessage takes a result and transform it in a string message
// PublishMessage sends the message to defined destination/set of consumer
// PublishExchange sends the message to all consumers
// SendMessage calls PrepareMessage and PublishMessage
// SendEvent calls PrepareMessage and PublishExchange
// Listen starts the message event loop and callback function on message receival
// GetGrammar returns the grammar for the input id
// SetGrammar saves the grammar
type Transport interface {
	Init(uid string)
	GetId() string
	GetQueueStatus(queue QueueType) (pending int, consumers int)
	GetProgress(uid string) (count int, ban int, match int)
	AddBan(uid string, nb int64)
	AddCount(uid string, nb int64)
	AddMatch(uid string, nb int64)
	SetBan(uid string, nb int64)
	SetCount(uid string, nb int64)
	SetMatch(uid string, nb int64)
	AddToBan(uid string, varUID string)
	GetToBan(uid string) (toBan []string)
	Clear(uid string)
	Close()
	PrepareMessage(logol.Result) string
	PublishMessage(queueName string, publish_msg string)
	PublishExchange(queueName string, publish_msg string)
	SendMessage(queue QueueType, data logol.Result) bool
	SendEvent(event MsgEvent) bool
	SendLog(msg string) bool
	Listen(queueListen QueueType, fn CallbackMessage)
	ListenLog(fn CallbackLog)
	GetGrammar(grammarId string) (g logol.Grammar, err bool)
	SetGrammar(grammar []byte, grammarId string) (err bool)
	IncrFlowStat(uid string, from string, to string)
	IncrDurationStat(uid string, variable string, duration int64)
	GetStats(uid string) Stats
}
