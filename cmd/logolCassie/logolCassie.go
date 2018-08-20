// Listen to events needing cassie library (needs cassiopee deb/rpm package)

package main


import (
        "log"
        listener "org.irisa.genouest/logol/lib/listener"
)


func main() {
    log.Printf("Listen to results")
    handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    handler.Cassie("test", nil)
}
