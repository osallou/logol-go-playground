// Listen to result events and write each match to output file
//
// Only 1 instance of logolResult should run for a search

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
