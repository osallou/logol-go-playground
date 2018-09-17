package logol

import (
    //"log"
    //"fmt"
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "time"
    logs "github.com/osallou/logol-go-playground/lib/log"
    logol "github.com/osallou/logol-go-playground/lib/types"
    // cassie "github.com/osallou/cassiopee-go"
    seq "github.com/osallou/logol-go-playground/lib/sequence"
    transport "github.com/osallou/logol-go-playground/lib/transport"
    //redis "github.com/go-redis/redis"
    //"github.com/satori/go.uuid"
)


var logger = logs.GetLogger("logol.message")

type MessageManager interface {
    Init(uid string, rch chan [][]logol.Match) // chan will work for results only, others will close chan at init
    Close()
    HandleMessage(logol.Result) bool
    Listen(queueListen transport.QueueType, fn transport.CallbackMessage)
}

type MessageResult struct {
    nbMatches int
    maxMatches int
    outfile *os.File
    outfileOpen bool
    msg msgManager
    msgLoaded bool
    uid string
    rch chan [][]logol.Match
}

func (m *MessageResult) Init(uid string, rch chan [][]logol.Match) {
    logger.Infof("init result %s", uid)
    m.nbMatches = 0
    m.maxMatches = 100
    os_maxMatches := os.Getenv("LOGOL_MAX_MATCH")
    if os_maxMatches != ""{
        maxMatches, err := strconv.Atoi(os_maxMatches)
        if err != nil {
            logger.Errorf("Invalid env variable LOGOL_MAX_MATCH, using default [100]")
            m.maxMatches = 100
        } else {
            m.maxMatches = maxMatches
        }
    }
    m.outfileOpen = false
    m.uid = uid
    if rch  != nil {
        m.rch = rch
    }
    t := transport.GetTransport(transport.QUEUE_RESULT)
    t.Init(m.uid)
}

func (m *MessageResult) Listen(queueListen transport.QueueType, fn transport.CallbackMessage) {
    t := transport.GetTransport(transport.QUEUE_RESULT)
    if t == nil {
        logger.Errorf("nul transport")
        os.Exit(1)
    }
    t.Listen(queueListen, fn)
}

func (m *MessageResult) Close(){
    logger.Infof("Closing result")
    if m.outfileOpen {
        m.outfile.Close()
    }
    if m.rch != nil {
        close(m.rch)
    }
}
func (m *MessageResult) HandleMessage(result logol.Result) (ok bool) {
    logger.Debugf("handle result message")
    if ! m.msgLoaded {
        t := transport.GetTransport(transport.QUEUE_RESULT)
        m.msg = NewMsgManager(m.uid, t)
        m.msgLoaded = true
    }

    if ! m.outfileOpen {
        m.outfileOpen = true
        outFilePath := "logol." + m.uid + ".out"
        if result.Outfile != "" {
            outFilePath = result.Outfile
        }
        logger.Infof("Create output file %s", outFilePath)
        outfile, err := os.Create(outFilePath)
        if err != nil {
        logger.Errorf("Failed to open output file")
        return false
        }
        m.outfile = outfile
    }
    json_msg, err := json.Marshal(result)
    logger.Errorf("Res: %s", json_msg)
    if err != nil {
        logger.Errorf("Failed to get message")
        return false
    }
    logger.Debugf("Match for job %s", result.Uid)
    matchOk := logol.CheckMatches(result.Matches, 0, true)
    if ! matchOk {
        m.msg.Transport.AddBan(result.Uid, 1)
        return false
    }
    for i:=0;i<len(result.PrevMatches);i++ {
        matchOk := logol.CheckMatches(result.PrevMatches[i], 0, true)
        if ! matchOk {
            m.msg.Transport.AddBan(result.Uid, 1)
            return false
        }
    }
    m.nbMatches += 1
    if m.nbMatches <= m.maxMatches {
        m.msg.Transport.AddMatch(result.Uid, 1)
        allMatches := append(result.PrevMatches, result.Matches)
        matches, _ := json.Marshal(allMatches)
        logger.Infof("Number of matches: %d", m.nbMatches)
        fmt.Fprintln(m.outfile, "", string(matches))
        if m.rch != nil {
            m.rch <- allMatches
        }
        logger.Debugf("%s", matches)
    }else {
        logger.Infof("Max results reached [%d], waiting to end...", m.maxMatches)
        m.msg.Transport.AddBan(result.Uid, 1)
    }
    return true
}


type MessageAnalyse struct {
    searchUtilsLoaded bool
    grammars map[string]logol.Grammar
    msg msgManager
    msgLoaded bool
    uid string
}

func (m *MessageAnalyse) Listen(queueListen transport.QueueType, fn transport.CallbackMessage) {
    t := transport.GetTransport(transport.QUEUE_MESSAGE)
    if t == nil {
        logger.Errorf("nul transport")
        os.Exit(1)
    }
    t.Listen(queueListen, fn)
}

func (m *MessageAnalyse) Init(uid string, rch chan [][]logol.Match) {
    logger.Infof("Init message analyse %s", uid)
    m.msgLoaded = false
    m.searchUtilsLoaded = false
    m.grammars = make(map[string]logol.Grammar)
    m.uid = uid
    if rch != nil {
        close(rch)
    }
    t := transport.GetTransport(transport.QUEUE_MESSAGE)
    t.Init(m.uid)
}

func (m *MessageAnalyse) Close()() {
    logger.Infof("Closing analyse")
}
func (m *MessageAnalyse) HandleMessage(result logol.Result) (ok bool) {
    //json_resmsg, _ := json.Marshal(result)
    //logger.Infof("Handle analyse message %s", json_resmsg)
    logger.Infof("Handle Analyse Message %s", result.Uid)

    if ! m.msgLoaded {
        t := transport.GetTransport(transport.QUEUE_MESSAGE)
        m.msg = NewMsgManager(m.uid, t)
        m.msgLoaded = true
    }

    g, ok := m.grammars[result.Uid]
    if ! ok {
        logger.Debugf("Load grammar not in cache, loading %s", result.Uid)
        g, err := m.msg.Transport.GetGrammar("logol:" + result.Uid + ":grammar")
        if err {
            logger.Errorf("Failed to get grammar %s", result.Uid)
            m.msg.Transport.AddBan(result.Uid, 1)
            return false
        }
        m.msg.Grammar = g
        m.grammars[result.Uid] = g
    }else {
        logger.Debugf("Load grammar from cache %s", result.Uid)
        m.msg.Grammar = g
    }

    if ! m.searchUtilsLoaded {
        m.msg.SearchUtils = seq.NewSearchUtils(m.msg.Grammar.Sequence)
        m.searchUtilsLoaded = true
    }

    logger.Debugf("Received message: %s", result.MsgTo)
    //json_msg, _ := json.Marshal(result)
    //logger.Debugf("#DEBUG# %s", json_msg)
    now := time.Now()
    start_time := now.UnixNano()
    now = time.Now()
    logger.Debugf("Received:Model:%s:Variable:%s", result.Model, result.ModelVariable)
    m.msg.handleMessage(result)
    end_time := now.UnixNano()
    duration := end_time - start_time
    m.msg.Transport.IncrDurationStat(result.Uid, result.Model + "." +result.ModelVariable, duration)
    logger.Debugf("Duration: %d", duration)
    return true
}



type MessageCassie struct {
    //cassieIndexer cassie.CassieIndexer
    //indexerLoaded bool
    searchUtilsLoaded bool
    grammars map[string]logol.Grammar
    msg msgManager
    msgLoaded bool
    uid string
}

func (m *MessageCassie) Listen(queueListen transport.QueueType, fn transport.CallbackMessage) {
    t := transport.GetTransport(transport.QUEUE_CASSIE)
    if t == nil {
        logger.Errorf("nul transport")
        os.Exit(1)
    }
    t.Listen(queueListen, fn)
}

func (m *MessageCassie) Init(uid string, rch chan [][]logol.Match) {
    logger.Infof("Init cassie %s", uid)
    m.msgLoaded = false
    //m.indexerLoaded = false
    m.searchUtilsLoaded = false
    m.grammars = make(map[string]logol.Grammar)
    m.uid = uid
    if rch != nil {
        close(rch)
    }
    t := transport.GetTransport(transport.QUEUE_CASSIE)
    t.Init(m.uid)
}
func (m *MessageCassie) Close() {
    logger.Infof("Closing cassie")
    //cassieIndexer := seq.GetCassieIndexer("")
    //cassie.DeleteCassieIndexer(*cassieIndexer)
}
func (m *MessageCassie) TreatMessage(result logol.Result){
    //json_msg, _ :=  json.Marshal(result)
    //logger.Infof("Received and should treat %s", json_msg)
    m.msg.handleMessage(result)
}
func (m *MessageCassie) HandleMessage(result logol.Result) (ok bool) {
    // Get grammar
    logger.Infof("Handle Cassie Message %s", result.Uid)

    if ! m.msgLoaded {
        t := transport.GetTransport(transport.QUEUE_CASSIE)
        m.msg = NewMsgManager(m.uid, t)
        m.msg.IsCassie = true
        m.msgLoaded = true
    }

    g, ok := m.grammars[result.Uid]
    if ! ok {
        logger.Debugf("Load grammar not in cache, loading %s", result.Uid)
        g, err := m.msg.Transport.GetGrammar("logol:" + result.Uid + ":grammar")
        if err {
            logger.Errorf("Failed to get grammar %s", result.Uid)
            m.msg.Transport.AddBan(result.Uid, 1)
            return false
        }
        m.msg.Grammar = g
        m.grammars[result.Uid] = g
    }else {
        logger.Debugf("Load grammar from cache %s", result.Uid)
        m.msg.Grammar = g
    }

    if ! m.searchUtilsLoaded {
        m.msg.SearchUtils = seq.NewSearchUtils(m.msg.Grammar.Sequence)
        m.searchUtilsLoaded = true
    }

    logger.Debugf("Received message: %s", result.MsgTo)

    now := time.Now()
    start_time := now.UnixNano()
    now = time.Now()
    logger.Debugf("Received:Model:%s:Variable:%s", result.Model, result.ModelVariable)
    m.TreatMessage(result)
    end_time := now.UnixNano()
    duration := end_time - start_time
    logger.Debugf("Duration: %d", duration)
    m.msg.Transport.IncrDurationStat(result.Uid, result.Model + "." +result.ModelVariable, duration)
    return true
}
