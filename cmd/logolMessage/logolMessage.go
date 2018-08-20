// Listen to events to find a variable

package main

import (
        "fmt"
        //"log"
        //"gopkg.in/yaml.v2"
        "github.com/streadway/amqp"
        //logol "org.irisa.genouest/logol/lib/types"
        msgHandler "org.irisa.genouest/logol/lib/listener"
)


func main() {
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

    handler := msgHandler.NewMsgHandler("localhost", 5672, "guest", "guest")
    handler.Listen("test", nil)
}
