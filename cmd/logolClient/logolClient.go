// Test client for logol

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	//"strconv"
	"syscall"
	"time"

	//"github.com/streadway/amqp"
	//msgHandler "org.irisa.genouest/logol/lib/listener"
	//redis "github.com/go-redis/redis"
	message "github.com/osallou/logol-go-playground/lib/message"
	transport "github.com/osallou/logol-go-playground/lib/transport"
	logol "github.com/osallou/logol-go-playground/lib/types"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"

	// msg "org.irisa.genouest/logol/lib/message"
	"github.com/namsral/flag"
	logs "github.com/osallou/logol-go-playground/lib/log"
	utils "github.com/osallou/logol-go-playground/lib/utils"
)

var logger = logs.GetLogger("logol.client")

func main() {
	var maxpatternlen int64
	var mode int64
	var standalone int64
	var grammarFile string
	var sequenceFile string
	var uid string
	var outfile string
	var version bool
	var nbAnalyseProc int64

	flag.Int64Var(&maxpatternlen, "maxpatternlen", 1000, "Maximum size of patterns to search")
	flag.Int64Var(&nbAnalyseProc, "procs", 1, "number of process to start")
	flag.Int64Var(&mode, "mode", 0, "Mode: 0=DNA, 1=RNA, 2=Protei")
	flag.Int64Var(&standalone, "standalone", 0, "Run in standalone mode, 0: multi process, 1: standalone")
	flag.StringVar(&grammarFile, "grammar", "grammar.txt", "Grammar file path")
	flag.StringVar(&uid, "uid", "run", "Unique identifier (will create result file logol.*uid*.out)")
	flag.StringVar(&sequenceFile, "sequence", "", "Sequence file path")
	flag.StringVar(&outfile, "out", "", "Output file path")
	flag.BoolVar(&version, "version", false, "Get version info")
	flag.Parse()
	logger.Infof("option maxpatternlen: %d", maxpatternlen)
	logger.Infof("option mode: %d", mode)

	if version {
		fmt.Printf("Version: %s\nBuild: %s\nGit commit: %s\n", utils.Version, utils.Buildstamp, utils.Githash)
		return
	}

	if _, err := os.Stat(grammarFile); os.IsNotExist(err) {
		log.Fatalf("Grammar file %s does not exist", grammarFile)
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
	g.Options["MODE"] = mode
	if sequenceFile != "" {
		g.Sequence = sequenceFile
	}

	if _, err := os.Stat(g.Sequence); os.IsNotExist(err) {
		log.Fatalf("Sequence file %s does not exist", g.Sequence)
	}

	updatedGrammar, _ := yaml.Marshal(&g)

	modelTo := g.Run[0].Model
	modelVariablesTo := g.Models[modelTo].Start

	var t transport.Transport
	t = transport.NewTransportRabbit()
	t.Init(uid)

	data := logol.NewResult()
	jobuid := uuid.Must(uuid.NewV4())
	data.Uid = jobuid.String()
	data.Outfile = outfile
	logger.Infof("Launch job %s", jobuid.String())

	t.SetCount(data.Uid, 1)
	t.SetBan(data.Uid, 0)
	t.SetMatch(data.Uid, 0)
	t.SetGrammar(updatedGrammar, data.Uid)

	go func() {
		var tLog transport.Transport
		tLog = transport.NewTransportRabbit()
		tLog.Init(uid)
		tLog.ListenLog(func(data string) bool {
			logger.Warningf(data)
			return true
		})
		tLog.Close()
	}()

	if standalone == 1 {
		go func() {
			log.Printf("Start cassie manager")
			var mngr message.MessageManager
			mngr = &message.MessageCassie{}
			mngr.Init(uid, nil)
			mngr.Listen(transport.QUEUE_CASSIE, mngr.HandleMessage)
			mngr.Close()
		}()
		var i int64
		for i = 0; i < nbAnalyseProc; i++ {
			go func() {
				log.Printf("Start analyse manager")
				var mngr message.MessageManager
				mngr = &message.MessageAnalyse{}
				mngr.Init(uid, nil)
				mngr.Listen(transport.QUEUE_MESSAGE, mngr.HandleMessage)
				mngr.Close()
			}()
		}
		go func() {
			log.Printf("Start result manager")
			var mngr message.MessageManager
			mngr = &message.MessageResult{}
			mngr.Init(uid, nil)
			mngr.Listen(transport.QUEUE_RESULT, mngr.HandleMessage)
			mngr.Close()
		}()
	}

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
	go func() {
		<-c
		log.Printf("Interrupt signal, exiting")
		event := transport.MsgEvent{}
		event.Step = transport.STEP_END
		t.SendEvent(event)
		notOver = false

	}()

	for notOver {
		pending, consumers := t.GetQueueStatus(transport.QUEUE_MESSAGE)
		log.Printf("Pending standard analyses: %d, by %d consumers", pending, consumers)
		pending, consumers = t.GetQueueStatus(transport.QUEUE_CASSIE)
		log.Printf("Pending cassie analyses: %d, by %d consumers", pending, consumers)
		count, ban, matches := t.GetProgress(data.Uid)
		log.Printf("Count: %d, Ban: %d, Matches: %d", count, ban, matches)

		if matches+ban == count {
			log.Printf("Search is over, exiting...")
			event := transport.MsgEvent{}
			event.Step = transport.STEP_END
			t.SendEvent(event)
			notOver = false
		}
		time.Sleep(2000 * time.Millisecond)
	}

	stats := t.GetStats(data.Uid)
	json_stats, _ := json.Marshal(stats)
	log.Printf("Stats: %s", json_stats)
	utils.WriteFlowPlots(data.Uid, stats.Flow, stats.Duration)

	t.Close()
	t.Clear(data.Uid)
}
