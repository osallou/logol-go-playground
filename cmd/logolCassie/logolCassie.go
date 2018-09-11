// Listen to events needing cassie library (needs cassiopee deb/rpm package)

package main


import (
        "log"
        //"os"
        message "github.com/osallou/logol-go-playground/lib/message"
        transport "github.com/osallou/logol-go-playground/lib/transport"
        "github.com/namsral/flag"
)


func main() {
    var uid string
    flag.StringVar(&uid, "uid", "run", "run identifier, same as logolClient")
    flag.Parse()

    log.Printf("Listen to cassie")
    /*
    uid := "test"
    os_uid := os.Getenv("LOGOL_UID")
    if os_uid != "" {
        uid = os_uid
    }*/

    //handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    //handler.Cassie("test", nil)
    var mngr message.MessageManager
    mngr = &message.MessageCassie{}
    //rch := make(chan [][]logol.Match)
    mngr.Init(uid, nil)
    mngr.Listen(transport.QUEUE_CASSIE, mngr.HandleMessage)
    mngr.Close()
}
