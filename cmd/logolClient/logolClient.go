// Test client for logol

package main

import (
        //"fmt"
        "log"
        //"encoding/json"
        "io/ioutil"
        "os"
        "os/signal"
        //"strconv"
        "syscall"
        "time"
        //"github.com/streadway/amqp"
        //msgHandler "org.irisa.genouest/logol/lib/listener"
        //redis "github.com/go-redis/redis"
        "gopkg.in/yaml.v2"
        "github.com/satori/go.uuid"
        logol "org.irisa.genouest/logol/lib/types"
        transport "org.irisa.genouest/logol/lib/transport"
        // msg "org.irisa.genouest/logol/lib/message"
        logs "org.irisa.genouest/logol/lib/log"
        "github.com/namsral/flag"
)

var logger = logs.GetLogger("logol.client")

func main() {
    var maxpatternlen int64
    flag.Int64Var(&maxpatternlen, "maxpatternlen", 1000, "Maximum size of patterns to search")
    flag.Parse()
    logger.Infof("option maxpatternlen: %d", maxpatternlen)


    uid := "test"
    os_uid := os.Getenv("LOGOL_UID")
    if os_uid != "" {
        uid = os_uid
    }
    grammarFile := "grammar.txt"
    osGrammar := os.Getenv("LOGOL_GRAMMAR")
    if osGrammar != "" {
        grammarFile = osGrammar
    }
    grammar, _ := ioutil.ReadFile(grammarFile)
    err, g := logol.LoadGrammar([]byte(grammar))
    if err != nil {
            log.Fatalf("error: %v", err)
    }

    if g.Options == nil {
        g.Options = make(map[string]int64)
    }
    g.Options["MAX_PATTERN_LENGTH"] = maxpatternlen
    updatedGrammar, _ := yaml.Marshal(&g)


    modelTo := g.Run[0].Model
    modelVariablesTo := g.Models[modelTo].Start

    var t transport.Transport
    t = transport.NewTransportRabbit()
    t.Init(uid)

    data := logol.NewResult()
    jobuid := uuid.Must(uuid.NewV4())
    data.Uid = jobuid.String()
    logger.Infof("Launch job %s", jobuid.String())

    t.SetCount(data.Uid, 1)
    t.SetBan(data.Uid, 0)
    t.SetMatch(data.Uid, 0)
    t.SetGrammar(updatedGrammar, data.Uid)

    for i := 0; i < len(modelVariablesTo); i++ {
        modelVariableTo := modelVariablesTo[i]
        data.MsgTo = "logol-" + modelTo + "-" + modelVariableTo
        data.Model = modelTo
        data.ModelVariable = modelVariableTo
        data.Spacer = true
        data.RunIndex = 0
        t.SendMessage(transport.QUEUE_MESSAGE, data)
    }

    notOver := true

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func(){
        <- c
        log.Printf("Interrupt signal, exiting")
        event := transport.MsgEvent{}
        event.Step = transport.STEP_END
        t.SendEvent(event)
        notOver = false

    }()

    for notOver {
        count, ban, matches := t.GetProgress(data.Uid)
        log.Printf("Count: %d, Ban: %d, Matches: %d", count, ban, matches)
        if matches + ban == count {
            log.Printf("Search is over, exiting...")
            event := transport.MsgEvent{}
            event.Step = transport.STEP_END
            t.SendEvent(event)
            notOver = false
        }
        time.Sleep(2000 * time.Millisecond)
    }


    t.Close()
    t.Clear(data.Uid)
}
