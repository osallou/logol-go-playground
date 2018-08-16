package main


import (
        "fmt"
        "log"
        "encoding/json"
        "os"
        "os/signal"
        "strconv"
        "syscall"
        "time"
        //"gopkg.in/yaml.v2"
        "github.com/streadway/amqp"
        msgHandler "org.irisa.genouest/logol/lib/listener"
        redis "github.com/go-redis/redis"
        "github.com/satori/go.uuid"
        logol "org.irisa.genouest/logol/lib/types"
)

const grammar = `
morphisms:
  - foo:
    - a
    - g
models:
  mod2:
    comment: 'mod2(+R2)'
    start: 'var1'
    param:
      - R2
    vars:
        var1:
            value: null
            string_constraints:
                content: 'R2'
            next: null


  mod1:
   comment: 'mod1(-R1)'
   param:
     - 'R1'
   start: 'var1'
   vars:
     var1:
         value: 'cc'
         next:
           - var2
     var2:
         value: 'aaa'
         string_constraints:
             saveas: 'R1'
         next:
          - var3
          - var4
     var3:
         value: null
         string_constraints:
           content: 'R1'
         next:
           - var5
     var4:
         comments: 'mod2(+R1)'
         value: null
         model:
             name: 'mod2'
             param:
               - R1
         next:
           - var5
     var5:
         value: 'cgt'
         next: null

run:
 - model: mod1
   param:
     - R8
 - model: mod2
   param:
     - R8

sequence: sequence.txt
`

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })


    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        "guest", "guest", "localhost", 5672)
    conn, _ := amqp.Dial(connUrl)
    defer conn.Close()
    ch, _ := conn.Channel()
    _, _ = ch.QueueDeclare(
      "logol-analyse-test", // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )
    ch.ExchangeDeclare(
      "logol-event-exchange-test", // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    u1 := uuid.Must(uuid.NewV4())

    jobuid := uuid.Must(uuid.NewV4())
    log.Printf("Launch job %s", jobuid.String())

    publish_msg := amqp.Publishing{}
    publish_msg.Body = []byte(u1.String())

    data := logol.NewResult()
    data.Uid = jobuid.String()
    err, g := logol.LoadGrammar([]byte(grammar))
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    modelTo := g.Run[0].Model
    modelVariableTo := g.Models[modelTo].Start
    data.MsgTo = "logol-" + modelTo + "-" + modelVariableTo
    data.Model = modelTo
    data.ModelVariable = modelVariableTo
    data.Spacer = true
    data.RunIndex = 0

    redisClient.Set("logol:" + data.Uid + ":grammar", grammar, 0)

    redisClient.Set("logol:" + data.Uid + ":count", 1, 0).Err()
    redisClient.Set("logol:" + data.Uid + ":match", 0, 0).Err()
    redisClient.Set("logol:" + data.Uid + ":ban", 0, 0).Err()


    json_msg, _ := json.Marshal(data)
    redisClient.Set(u1.String(), json_msg, 0).Err()
    log.Printf("Send message %s, %s", u1.String(), string(publish_msg.Body))

    ch.Publish(
        "", // exchange
        "logol-analyse-test", // key
        false, // mandatory
        false, // immediate
        publish_msg,
    )

    notOver := true

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func(){
        <- c
        log.Printf("Interrupt signal, exiting")
        publish_msg := amqp.Publishing{}
        msg := logol.Event{}
        msg.Step = msgHandler.STEP_END
        exit_msg, _ := json.Marshal(msg)
        publish_msg.Body = []byte(exit_msg)
        ch.Publish(
            "logol-event-exchange-test", // exchange
            "", // key
            false, // mandatory
            false, // immediate
            publish_msg,
        )
        notOver = false

    }()


    for notOver {
        rcount, _ := redisClient.Get("logol:" + data.Uid + ":count").Result()
        count, _ := strconv.Atoi(rcount)
        rban, _ := redisClient.Get("logol:" + data.Uid + ":ban").Result()
        ban, _ := strconv.Atoi(rban)
        rmatches, _ := redisClient.Get("logol:" + data.Uid + ":match").Result()
        matches, _ := strconv.Atoi(rmatches)
        log.Printf("Count: %d, Ban: %d, Matches: %d", count, ban, matches)
        if matches + ban == count {
            log.Printf("Search is over, exiting...")
            publish_msg := amqp.Publishing{}
            msg := logol.Event{}
            msg.Step = msgHandler.STEP_END
            exit_msg, _ := json.Marshal(msg)
            publish_msg.Body = []byte(exit_msg)
            ch.Publish(
                "logol-event-exchange-test", // exchange
                "", // key
                false, // mandatory
                false, // immediate
                publish_msg,
            )
            notOver = false
        }
        time.Sleep(2000 * time.Millisecond)
    }
    redisClient.Del("logol:" + data.Uid + ":count")
    redisClient.Del("logol:" + data.Uid + ":match")
    redisClient.Del("logol:" + data.Uid + ":ban")
    ch.ExchangeDelete("logol-event-exchange-test", false, false)
    ch.QueueDelete("logol-analyse-test", false, false, false)
    ch.QueueDelete("logol-result-test", false, false, false)
    ch.QueueDelete("logol-cassie-test", false, false, false)
}
