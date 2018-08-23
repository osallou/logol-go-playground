// Listen to result events and write each match to output file
//
// Only 1 instance of logolResult should run for a search

package main

import (
        listener "org.irisa.genouest/logol/lib/listener"
        logol "org.irisa.genouest/logol/lib/types"
        logs "org.irisa.genouest/logol/lib/log"
)

var logger = logs.GetLogger("logol.result")

func main() {
    logger.Infof("Listen to results")
    resChan := make(chan [][]logol.Match)
    handler := listener.NewMsgHandler("localhost", 5672, "guest", "guest")
    go handler.Results(resChan, "test", nil)
    nbResults := 0
    for _ = range resChan {
        nbResults += 1
        logger.Infof("NbResults:%d", nbResults)
    }
}
