package main


import (
        "log"
        listener "org.irisa.genouest/logol/lib/listener"
)


func main() {
    log.Printf("Listen to results")
    handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    handler.Results("test", nil)
}
