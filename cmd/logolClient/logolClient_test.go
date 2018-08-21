// Test client for logol

package main

import (
        "path/filepath"
        "fmt"
        "log"
        "encoding/json"
        "io/ioutil"
        "testing"
        //"gopkg.in/yaml.v2"
        "github.com/streadway/amqp"
        msgHandler "org.irisa.genouest/logol/lib/listener"
        redis "github.com/go-redis/redis"
        "github.com/satori/go.uuid"
        logol "org.irisa.genouest/logol/lib/types"
)



func stop(ch *amqp.Channel) {
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
}

func startGrammar(resChan chan [][]logol.Match, grammarFile string) ([][] logol.Match){
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

    grammar, _ := ioutil.ReadFile(grammarFile)
    err, g := logol.LoadGrammar([]byte(grammar))
    if err != nil {
            log.Fatalf("error: %v", err)
    }

    modelTo := g.Run[0].Model
    modelVariablesTo := g.Models[modelTo].Start
    redisClient.Set("logol:" + data.Uid + ":grammar", grammar, 0)
    redisClient.Set("logol:" + data.Uid + ":count", len(modelVariablesTo), 0).Err()
    redisClient.Set("logol:" + data.Uid + ":match", 0, 0).Err()
    redisClient.Set("logol:" + data.Uid + ":ban", 0, 0).Err()

    testMsgHandler := msgHandler.NewMsgHandler("localhost", 5672, "guest", "guest")
    go testMsgHandler.Listen("test", nil)
    cassieHandler := msgHandler.NewMsgHandler("localhost", 5672, "guest", "guest")
    go cassieHandler.Cassie("test", nil)
    resultHandler := msgHandler.NewMsgHandler("localhost", 5672, "guest", "guest")
    go resultHandler.Results(resChan, "test", nil)

    for i := 0; i < len(modelVariablesTo); i++ {
        modelVariableTo := modelVariablesTo[i]
        data.MsgTo = "logol-" + modelTo + "-" + modelVariableTo
        data.Model = modelTo
        data.ModelVariable = modelVariableTo
        data.Spacer = true
        data.RunIndex = 0

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
    }

    stopSent := false
    nbResults := 0
    firstResult := make([][]logol.Match, 0)
    for result := range resChan {
        nbResults += 1
        if nbResults == 1 {
            firstResult = result
        }
        if ! stopSent {
            stop(ch)
            stopSent = true
        }
    }


    redisClient.Del("logol:" + data.Uid + ":count")
    redisClient.Del("logol:" + data.Uid + ":match")
    redisClient.Del("logol:" + data.Uid + ":ban")
    ch.ExchangeDelete("logol-event-exchange-test", false, false)
    ch.QueueDelete("logol-analyse-test", false, false, false)
    ch.QueueDelete("logol-result-test", false, false, false)
    ch.QueueDelete("logol-cassie-test", false, false, false)

    return firstResult
}
func TestGrammar(t *testing.T) {
    //handler := Handler{}
    grammar := filepath.Join("testdata", "grammar.txt")
    resChan := make(chan [][]logol.Match)
    result := startGrammar(resChan, grammar)
    json_msg, _ := json.Marshal(result)
    log.Printf("Result: %s", json_msg)
    if len(result) != 2 {
        t.Errorf("Invalid number of model")
    }
    model1 := result[0]
    var1 := model1[0]
    if var1.Start != 2 && var1.End != 4 {
        t.Errorf("Invalid result: %s", json_msg)
    }

}

func TestNegConstraint(t *testing.T) {
    //handler := Handler{}
    log.Printf("Test negative constraint")
    grammar := filepath.Join("testdata", "negative_constraint.txt")
    resChan := make(chan [][]logol.Match)
    result := startGrammar(resChan, grammar)
    json_msg, _ := json.Marshal(result)
    log.Printf("Result: %s", json_msg)
    if len(result) != 1 {
        t.Errorf("Invalid number of model")
    }
    model1 := result[0]
    var1 := model1[0]
    if var1.Start != 4 && var1.End != 10 {
        t.Errorf("Invalid result: %s", json_msg)
    }
    var2 := model1[1]
    if var2.Start != 10 && var2.End != 15 {
        t.Errorf("Invalid result: %s", json_msg)
    }
}
