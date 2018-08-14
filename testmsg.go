package main


import (
        "fmt"
        "log"
        "encoding/json"
        //"gopkg.in/yaml.v2"
        "github.com/streadway/amqp"
        //msgHandler "org.irisa.genouest/logol/lib/listener"
        redis "github.com/go-redis/redis"
        "github.com/satori/go.uuid"
        logol "org.irisa.genouest/logol/lib/types"
)


func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })


    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        "guest", "guest", "localhost", 5672)
    conn, _ := amqp.Dial(connUrl)
    ch, _ := conn.Channel()
    _, _ = ch.QueueDeclare(
      "logol-analyse-test", // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    u1 := uuid.Must(uuid.NewV4())


    redisClient.Set("logol:count", 1, 0).Err()

    redisClient.Set("logol:match", 0, 0).Err()

    redisClient.Set("logol:ban", 0, 0).Err()



    publish_msg := amqp.Publishing{}
    publish_msg.Body = []byte(u1.String())

    data := logol.NewResult()
    data.MsgTo = "logol-mod1-var1"
    data.Model = "mod1"
    data.ModelVariable = "var1"
    data.Spacer = true

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
