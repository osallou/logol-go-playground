package main


import (
        "log"
        "os"
        message "org.irisa.genouest/logol/lib/message"
        transport "org.irisa.genouest/logol/lib/transport"
)


func main() {
    log.Printf("Listen to results")
    uid := "test"
    os_uid := os.Getenv("LOGOL_UID")
    if os_uid != "" {
        uid = os_uid
    }
    //handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    //handler.Cassie("test", nil)
    var mngr message.MessageManager
    mngr = &message.MessageResult{}
    mngr.Init(uid, nil)
    mngr.Listen(transport.QUEUE_RESULT, mngr.HandleMessage)
    mngr.Close()
}
