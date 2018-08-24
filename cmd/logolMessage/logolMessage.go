// Listen to events to find a variable

package main


import (
        "log"
        "os"
        transport "org.irisa.genouest/logol/lib/transport"
        message "org.irisa.genouest/logol/lib/message"
)


func main() {
    log.Printf("Listen to analyse")
    uid := "test"
    os_uid := os.Getenv("LOGOL_UID")
    if os_uid != "" {
        uid = os_uid
    }
    //handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    //handler.Cassie("test", nil)
    var mngr message.MessageManager
    mngr = &message.MessageAnalyse{}
    mngr.Init(uid, nil)
    mngr.Listen(transport.QUEUE_MESSAGE, mngr.HandleMessage)
    mngr.Close()
}
