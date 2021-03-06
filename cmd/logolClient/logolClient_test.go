// Test client for logol

package main

import (
	"os"
	"path/filepath"

	"encoding/json"
	"io/ioutil"
	"log"
	"testing"

	message "github.com/osallou/logol-go-playground/lib/message"
	transport "github.com/osallou/logol-go-playground/lib/transport"
	logol "github.com/osallou/logol-go-playground/lib/types"
	"github.com/satori/go.uuid"
)

func stop(t transport.Transport) {
	event := transport.MsgEvent{}
	event.Step = transport.STEP_END
	t.SendEvent(event)
	os.Remove("logol." + t.GetID() + ".out")
}

func startGrammar(resChan chan [][]logol.Match, grammarFile string) [][]logol.Match {
	//uid := "test"
	uid := uuid.Must(uuid.NewV4()).String()
	grammar, _ := ioutil.ReadFile(grammarFile)
	err, g := logol.LoadGrammar([]byte(grammar))
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	var t transport.Transport
	t = transport.NewTransportRabbit()
	t.Init(uid)

	data := logol.NewResult()
	jobuid := uuid.Must(uuid.NewV4())
	data.Uid = jobuid.String()

	t.SetCount(data.Uid, 1)
	t.SetBan(data.Uid, 0)
	t.SetMatch(data.Uid, 0)
	t.SetGrammar(grammar, data.Uid)

	go func() {
		log.Printf("Start cassie manager")
		var mngr message.MessageManager
		mngr = &message.MessageCassie{}
		mngr.Init(uid, nil)
		mngr.Listen(transport.QUEUE_CASSIE, mngr.HandleMessage)
		mngr.Close()
	}()
	go func() {
		log.Printf("Start analyse manager")
		var mngr message.MessageManager
		mngr = &message.MessageAnalyse{}
		mngr.Init(uid, nil)
		mngr.Listen(transport.QUEUE_MESSAGE, mngr.HandleMessage)
		mngr.Close()
	}()
	go func() {
		log.Printf("Start result manager")
		var mngr message.MessageManager
		mngr = &message.MessageResult{}
		mngr.Init(uid, resChan)
		//mngr.Init(uid, nil)
		mngr.Listen(transport.QUEUE_RESULT, mngr.HandleMessage)
		mngr.Close()
	}()

	modelTo := g.Run[0].Model
	modelVariablesTo := g.Models[modelTo].Start

	for i := 0; i < len(modelVariablesTo); i++ {
		modelVariableTo := modelVariablesTo[i]
		data.MsgTo = "logol-" + modelTo + "-" + modelVariableTo
		data.Model = modelTo
		data.ModelVariable = modelVariableTo
		data.Spacer = true
		data.RunIndex = 0
		t.SendMessage(transport.QUEUE_MESSAGE, data)
	}

	stopSent := false
	nbResults := 0
	firstResult := make([][]logol.Match, 0)
	log.Printf("Wait for results now....")

	for result := range resChan {
		nbResults++
		if nbResults == 1 {
			firstResult = result
		}
		count, ban, matches := t.GetProgress(data.Uid)
		log.Printf("Progress %d %d %d", count, ban, matches)
		if matches+ban >= count {
			if !stopSent {
				stop(t)
				stopSent = true
			}
		}
	}

	t.Clear(data.Uid)
	t.Close()

	return firstResult
}

func TestGrammar(t *testing.T) {
	log.Printf("Test grammar")
	//handler := Handler{}
	grammar := filepath.Join("testdata", "grammar.txt")
	resChan := make(chan [][]logol.Match)
	result := startGrammar(resChan, grammar)
	jsonMsg, _ := json.Marshal(result)
	log.Printf("Result: %s", jsonMsg)
	if len(result) != 2 {
		t.Errorf("Invalid number of model")
	}
	model1 := result[0]
	var1 := model1[0]
	if var1.Start != 2 && var1.End != 4 {
		t.Errorf("Invalid result: %s", jsonMsg)
	}

}

func TestNegConstraint(t *testing.T) {
	//handler := Handler{}
	log.Printf("Test negative constraint")
	grammar := filepath.Join("testdata", "negative_constraint.txt")
	resChan := make(chan [][]logol.Match)
	result := startGrammar(resChan, grammar)
	jsonMsg, _ := json.Marshal(result)
	log.Printf("Result: %s", jsonMsg)
	if len(result) != 1 {
		t.Errorf("Invalid number of model")
	}
	model1 := result[0]
	var1 := model1[0]
	if var1.Start != 4 && var1.End != 10 {
		t.Errorf("Invalid result: %s", jsonMsg)
	}
	var2 := model1[1]
	if var2.Start != 10 && var2.End != 15 {
		t.Errorf("Invalid result: %s", jsonMsg)
	}
}

func TestGrammarNot(t *testing.T) {
	log.Printf("Test grammar")
	//handler := Handler{}
	grammar := filepath.Join("testdata", "grammar_not.txt")
	resChan := make(chan [][]logol.Match)
	result := startGrammar(resChan, grammar)
	jsonMsg, _ := json.Marshal(result)
	log.Printf("Result: %s", jsonMsg)
	if len(result) == 0 || len(result[0]) == 0 {
		t.Errorf("should have found a model")
	}

}
