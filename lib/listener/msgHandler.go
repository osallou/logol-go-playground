package logol

import (
  "fmt"
  "log"
  "os"
  "encoding/json"
  "strconv"
  "time"
  "github.com/streadway/amqp"
  logol "org.irisa.genouest/logol/lib/types"
  cassie "org.irisa.genouest/cassiopee"
)

const STEP_NONE int = -1
const STEP_PRE int = 0
const STEP_POST int = 1
const STEP_END int = 2
const STEP_BAN int = 3
const STEP_CASSIE int = 4
const STEP_YETTOBEDEFINED int = 5


func failOnError(err error, msg string) {
  if err != nil {
    log.Fatalf("%s: %s", msg, err)
    panic(fmt.Sprintf("%s: %s", msg, err))
  }
}

type MsgEvent struct {
    Step  int
}


type MsgHandler struct {
    Hostname string
    Port int
    User string
    Password string
}

type MsgCallback func([]byte) bool

func NewMsgHandler(host string, port int, user string, password string) MsgHandler {
    msgHandler := MsgHandler{}
    msgHandler.Hostname = host
    msgHandler.Port = 5672
    msgHandler.User = "guest"
    msgHandler.Password = "guest"
    if user != "" {
        msgHandler.User = user
        msgHandler.Password = password
    }
    return msgHandler
}

func (h MsgHandler) Cassie(queueName string, fn MsgCallback) {
    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        h.User, h.Password, h.Hostname, h.Port)
    conn, err := amqp.Dial(connUrl)
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, rqerr := ch.QueueDeclare(
      "logol-cassie-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(rqerr, "Failed to declare a queue")

    err = ch.ExchangeDeclare(
      "logol-event-exchange-" + queueName, // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare an exchange")

    eventQueue, err := ch.QueueDeclare(
        "",
        false,   // durable
        false,   // delete when usused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        eventQueue.Name, // name,
        "", // key
        "logol-event-exchange-" + queueName,  // exchange
        false, // no-wait
        nil, // arguments
    )

    failOnError(err, "Failed to bind queue")

    err = ch.Qos(
      1,     // prefetch count
      0,     // prefetch size
      false, // global
    )
    failOnError(err, "Failed to set QoS")

    msgs, err := ch.Consume(
      q.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    events, err := ch.Consume(
      eventQueue.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    msgManager := NewMsgManager("localhost", ch, "test")

    grammars := make(map[string]logol.Grammar)

    forever := make(chan bool)

    go func() {
        //var cassieIndexer cassie.CassieIndexer = nil
        //msgManager.CassieManager = logol.NewCassieManager()
        var cassieIndexer cassie.CassieIndexer
        //defer cassie.DeleteCassieIndexer(msgManager.CassieManager.Indexer)
        indexerLoaded := false
        searchUtilsLoaded := false

        for d := range msgs {
            result, err := msgManager.get(string(d.Body[:]))
            if err != nil {
                log.Printf("Failed to get message")
                d.Ack(false)
                continue
            }
            // Get grammar
            g, ok := grammars[result.Uid]
            if ! ok {
                log.Printf("Load grammar not in cache, loading %s", result.Uid)
                grammar, err := msgManager.Client.Get("logol:" + result.Uid + ":grammar").Result()
                if grammar == "" {
                    log.Printf("Failed to get grammar %s", result.Uid)
                    msgManager.Client.Incr("logol:" + result.Uid + ":ban")
                    d.Ack(false)
                    continue
                }
                err, g := logol.LoadGrammar([]byte(grammar))
                if err != nil {
                        log.Fatalf("error: %v", err)
                }
                msgManager.Grammar = g
                grammars[result.Uid] = g
            }else {
                log.Printf("Load grammar from cache %s", result.Uid)
                msgManager.Grammar = g
            }

            if ! searchUtilsLoaded {
                msgManager.SearchUtils = msgManager.SetSearchUtils(msgManager.Grammar.Sequence)
                searchUtilsLoaded = true
            }


            if ! indexerLoaded {
                // TODO should reindex if pattern length is longer this time
                //msgManager.CassieManager.GetIndexer(msgManager.Grammar.Sequence)
                cassieIndexer = cassie.NewCassieIndexer(msgManager.Grammar.Sequence)
                cassieIndexer.SetMax_index_depth(1000)
                cassieIndexer.SetMax_depth(10000)
                cassieIndexer.SetDo_reduction(true)
                cassieIndexer.Index()
                indexerLoaded = true
            }
            log.Printf("Load cassie searcher")
            cassieSearcher := cassie.NewCassieSearch(cassieIndexer)
            cassieSearcher.SetMode(0)
            cassieSearcher.SetMax_subst(0)
            cassieSearcher.SetMax_indel(0)
            cassieSearcher.SetAmbiguity(false)
            msgManager.CassieManager = logol.Cassie{cassieIndexer, cassieSearcher,0}


            log.Printf("Received message: %s", result.MsgTo)
            // TODO to remove, for debug only
            json_msg, _ := json.Marshal(result)
            log.Printf("#DEBUG# %s", json_msg)
            now := time.Now()
            start_time := now.UnixNano()
            now = time.Now()
            log.Printf("Received:Model:%s:Variable:%s", result.Model, result.ModelVariable)
            msgManager.handleMessage(result)
            end_time := now.UnixNano()
            duration := end_time - start_time
            sendStats(result.Model, result.ModelVariable, duration)
            //cassie.DeleteCassieSearch(msgManager.CassieManager.Searcher)
            // json.Unmarshal([]byte(d.Body), &result)
            cassie.DeleteCassieSearch(cassieSearcher)
            d.Ack(false)
        }
        cassie.DeleteCassieIndexer(cassieIndexer)
    }()

    go func(ch chan bool) {
      for d := range events {
        log.Printf("Received an event: %s", d.Body)
        msgEvent := MsgEvent{}
        json.Unmarshal([]byte(d.Body), &msgEvent)
        switch msgEvent.Step {
            case STEP_END:
                log.Printf("Received exit request")
                d.Ack(false)
                //os.Exit(0)
                ch <- true
            default:
                d.Ack(false)
        }
      }
    }(forever)


    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever
    ch.Close()
}

func (h MsgHandler) Results(queueName string, fn MsgCallback) {
    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        h.User, h.Password, h.Hostname, h.Port)
    conn, err := amqp.Dial(connUrl)
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, rqerr := ch.QueueDeclare(
      "logol-result-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(rqerr, "Failed to declare a queue")

    err = ch.ExchangeDeclare(
      "logol-event-exchange-" + queueName, // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare an exchange")

    eventQueue, err := ch.QueueDeclare(
        "",
        false,   // durable
        false,   // delete when usused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        eventQueue.Name, // name,
        "", // key
        "logol-event-exchange-" + queueName,  // exchange
        false, // no-wait
        nil, // arguments
    )

    failOnError(err, "Failed to bind queue")

    err = ch.Qos(
      1,     // prefetch count
      0,     // prefetch size
      false, // global
    )
    failOnError(err, "Failed to set QoS")

    msgs, err := ch.Consume(
      q.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    events, err := ch.Consume(
      eventQueue.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    forever := make(chan bool)


    msgManager := NewMsgManager("localhost", ch, "test")

    nbMatches := 0
    maxMatches := 100
    os_maxMatches := os.Getenv("LOGOL_MAX_MATCH")
    if os_maxMatches != ""{
        maxMatches, err = strconv.Atoi(os_maxMatches)
        if err != nil {
            log.Printf("Invalid env variable LOGOL_MAX_MATCH, using default [100]")
            maxMatches = 100
        }
    }

    go func() {
        file, err := os.Create("logol." + queueName + ".out")
        failOnError(err, "Failed to create output file")
        defer file.Close()

        for d := range msgs {
            log.Printf("Received a message: %s", string(d.Body[:]))
            result, err := msgManager.get(string(d.Body[:]))

            json_msg, _ := json.Marshal(result)
            log.Printf("Res: %s", json_msg)
            if err != nil {
                log.Printf("Failed to get message")
                d.Ack(false)
                continue
            }
            log.Printf("Match for job %s", result.Uid)
            matchOk := logol.CheckMatches(result.Matches)
            if ! matchOk {
                msgManager.Client.Incr("logol:" + result.Uid + ":ban")
                d.Ack(false)
                continue
            }
            nbMatches += 1
            if nbMatches <= maxMatches {
                msgManager.Client.Incr("logol:" + result.Uid + ":match")
                allMatches := append(result.PrevMatches, result.Matches)
                matches, _ := json.Marshal(allMatches)
                fmt.Fprintln(file, "", string(matches))
                log.Printf("%s", matches)
            }else {
                log.Printf("Max results reached [%d], waiting to end...", maxMatches)
                msgManager.Client.Incr("logol:" + result.Uid + ":ban")
            }
            d.Ack(false)
        }
    }()

    go func(ch chan bool) {
      for d := range events {
        log.Printf("Received an event: %s", d.Body)
        msgEvent := MsgEvent{}
        json.Unmarshal([]byte(d.Body), &msgEvent)
        switch msgEvent.Step {
            case STEP_END:
                log.Printf("Received exit request")
                d.Ack(false)
                //os.Exit(0)
                ch <- true
            default:
                d.Ack(false)
        }
      }
    }(forever)


    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever

    ch.Close()
}


func (h MsgHandler) Listen(queueName string, fn MsgCallback) {
    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        h.User, h.Password, h.Hostname, h.Port)
    conn, err := amqp.Dial(connUrl)
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, err := ch.QueueDeclare(
      "logol-analyse-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    _, rqerr := ch.QueueDeclare(
      "logol-result-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(rqerr, "Failed to declare a queue")

    err = ch.ExchangeDeclare(
      "logol-event-exchange-" + queueName, // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare an exchange")

    eventQueue, err := ch.QueueDeclare(
        "",
        false,   // durable
        false,   // delete when usused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        eventQueue.Name, // name,
        "", // key
        "logol-event-exchange-" + queueName,  // exchange
        false, // no-wait
        nil, // arguments
    )

    failOnError(err, "Failed to bind queue")

    err = ch.Qos(
      1,     // prefetch count
      0,     // prefetch size
      false, // global
    )
    failOnError(err, "Failed to set QoS")

    msgs, err := ch.Consume(
      q.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    events, err := ch.Consume(
      eventQueue.Name, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    forever := make(chan bool)

    ban := false

    msgManager := NewMsgManager("localhost", ch, "test")

    grammars := make(map[string]logol.Grammar)

    /*
    err, g := logol.LoadGrammar([]byte(grammar))
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    msgManager.Grammar = g
    */

    go func() {
      searchUtilsLoaded := false
      for d := range msgs {
        log.Printf("Received a message: %s", string(d.Body[:]))
        if ban {
            d.Ack(false)
            continue

        } else {

            result, err := msgManager.get(string(d.Body[:]))
            if err != nil {
                log.Printf("Failed to get message")
                d.Ack(false)
                continue
            }
            // Get grammar
            g, ok := grammars[result.Uid]
            if ! ok {
                log.Printf("Load grammar not in cache, loading %s", result.Uid)
                grammar, err := msgManager.Client.Get("logol:" + result.Uid + ":grammar").Result()
                if grammar == "" {
                    log.Printf("Failed to get grammar %s", result.Uid)
                    msgManager.Client.Incr("logol:" + result.Uid + ":ban")
                    d.Ack(false)
                    continue
                }
                err, g := logol.LoadGrammar([]byte(grammar))
                if err != nil {
                        log.Fatalf("error: %v", err)
                }
                msgManager.Grammar = g
                grammars[result.Uid] = g
            }else {
                log.Printf("Load grammar from cache %s", result.Uid)
                msgManager.Grammar = g
            }

            if ! searchUtilsLoaded {
                //log.Printf("Init sequence lru")
                msgManager.SearchUtils = msgManager.SetSearchUtils(msgManager.Grammar.Sequence)
                searchUtilsLoaded = true
            }
            log.Printf("##### %d", msgManager.SearchUtils.SequenceHandler.Sequence.Size)


            log.Printf("Received message: %s", result.MsgTo)
            // TODO to remove, for debug only
            json_msg, _ := json.Marshal(result)
            log.Printf("#DEBUG# %s", json_msg)
            now := time.Now()
            start_time := now.UnixNano()
            now = time.Now()
            log.Printf("Received:Model:%s:Variable:%s", result.Model, result.ModelVariable)
            msgManager.handleMessage(result)
            end_time := now.UnixNano()
            duration := end_time - start_time
            sendStats(result.Model, result.ModelVariable, duration)
            // json.Unmarshal([]byte(d.Body), &result)
            d.Ack(false)

        }
      }
    }()

    go func(ch chan bool) {
      for d := range events {
        log.Printf("Received an event: %s", d.Body)
        msgEvent := MsgEvent{}
        json.Unmarshal([]byte(d.Body), &msgEvent)
        switch msgEvent.Step {
            case STEP_END:
                log.Printf("Received exit request")
                d.Ack(false)
                ch <- true
                //os.Exit(0)
            case STEP_BAN:
                ban = true
                d.Ack(false)
            default:
                d.Ack(false)
        }
      }
    }(forever)


    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever

    ch.Close()
}
