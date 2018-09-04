package main


import (
        "log"
        //"os"
        message "org.irisa.genouest/logol/lib/message"
        transport "org.irisa.genouest/logol/lib/transport"
        "github.com/namsral/flag"
)


func main() {
    log.Printf("Listen to results")
    var uid string
    flag.StringVar(&uid, "uid", "run", "run identifier, same as logolClient")
    flag.Parse()
    //handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    //handler.Cassie("test", nil)
    var mngr message.MessageManager
    mngr = &message.MessageResult{}
    mngr.Init(uid, nil)
    mngr.Listen(transport.QUEUE_RESULT, mngr.HandleMessage)
    mngr.Close()
}
