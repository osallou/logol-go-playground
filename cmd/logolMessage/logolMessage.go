// Listen to events to find a variable

package main


import (
        "log"
        //"os"
        transport "github.com/osallou/logol-go-playground/lib/transport"
        message "github.com/osallou/logol-go-playground/lib/message"
        "github.com/namsral/flag"
)


func main() {
    log.Printf("Listen to analyse")
    var uid string
    flag.StringVar(&uid, "uid", "run", "run identifier, same as logolClient")
    flag.Parse()
    //handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    //handler.Cassie("test", nil)
    var mngr message.MessageManager
    mngr = &message.MessageAnalyse{}
    mngr.Init(uid, nil)
    mngr.Listen(transport.QUEUE_MESSAGE, mngr.HandleMessage)
    mngr.Close()
}
