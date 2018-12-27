// Manage input messages

package logol

import (
	"encoding/json"
	"net/http"
	"os"

	//"log"
	"sort"
	"strconv"

	//"strings"
	seq "github.com/osallou/logol-go-playground/lib/sequence"
	transport "github.com/osallou/logol-go-playground/lib/transport"
	logol "github.com/osallou/logol-go-playground/lib/types"
	utils "github.com/osallou/logol-go-playground/lib/utils"

	//redis "github.com/go-redis/redis"
	//"github.com/streadway/amqp"
	"github.com/satori/go.uuid"
)

// Structure managing global access to different tools and information
type msgManager struct {
	Chuid       string
	IsCassie    bool
	Grammar     logol.Grammar
	SearchUtils seq.SearchUtils
	Transport   transport.Transport
}

// newMsgManager creates a new MsgManager instance with a Transport
func newMsgManager(uid string, t transport.Transport) msgManager {
	manager := msgManager{}
	manager.Chuid = uid
	manager.Transport = t
	return manager
}

// check for input/output params of model, and set them with current context variables
func (m msgManager) setParam(contextVars map[string]logol.Match, param []string) []logol.Match {
	var res []logol.Match
	for _, modelOutput := range param {
		logger.Debugf("Set param %s", modelOutput)
		cv, ok := contextVars[modelOutput]
		if !ok {
			// DEBUG json print for debug
			jsonVars, _ := json.Marshal(contextVars)
			logger.Debugf("Param not in contextVars: %s", jsonVars)
			m := logol.NewMatch()
			m.Id = modelOutput
			res = append(res, m)
		} else {
			logger.Debugf("Param in contextVars")
			res = append(res, cv)
		}
	}
	return res
}

// checks if an other main model is to be executed
func (m msgManager) hasNextModel(data logol.Result) bool {
	modelsToRun := len(m.Grammar.Run) - 1
	logger.Debugf("Other main models? %d vs %d", data.RunIndex, modelsToRun)
	if data.RunIndex < modelsToRun {
		return true
	}
	return false
}

// Go to next main model
//
// Takes as input the current model and current result
func (m msgManager) goNextModel(model string, data logol.Result) {
	modelTo := m.Grammar.Run[data.RunIndex+1].Model
	modelVariablesTo := m.Grammar.Models[modelTo].Start

	if m.Grammar.Run[data.RunIndex+1].Nomatch != "" {
		logger.Debugf("goNextModel: expect no match")
		data.ExpectNoMatchUID = m.getUID()
	}

	for i := 0; i < len(modelVariablesTo); i++ {
		if i > 0 {
			//m.Client.Incr("logol:" + data.Uid + ":count")
			m.Transport.AddCount(data.Uid, 1)
		}
		modelVariableTo := modelVariablesTo[i]
		logger.Debugf("Go to next main model %s:%s", modelTo, modelVariableTo)
		tmpResult := logol.NewResult()
		tmpResult.ExpectNoMatch = data.ExpectNoMatch
		//tmpResult.ExpectNoMatchUID = data.ExpectNoMatchUID
		tmpResult.Uid = data.Uid
		tmpResult.Outfile = data.Outfile
		//data.From = make([]string, 0)
		tmpResult.PrevMatches = append(data.PrevMatches, data.Matches)
		tmpResult.Matches = make([]logol.Match, 0)
		tmpResult.Spacer = true
		tmpResult.Position = 0
		tmpResult.YetToBeDefined = data.YetToBeDefined
		// Update params
		tmpContextVars := make(map[string]logol.Match)
		currentModelParams := m.Grammar.Run[data.RunIndex].Param
		for i, param := range currentModelParams {
			tmpContextVars[param] = data.ContextVars[len(data.ContextVars)-1][m.Grammar.Models[model].Param[i]]
		}

		if m.Grammar.Run[data.RunIndex+1].Nomatch != "" {
			noMatchVar := data.ContextVars[len(data.ContextVars)-1][m.Grammar.Run[data.RunIndex+1].Nomatch]
			tmpResult.ExpectNoMatch = true
			tmpResult.ExpectNoMatchVar = noMatchVar
			tmpResult.Spacer = false
			tmpResult.Position = noMatchVar.Start
			logger.Debugf("New model expects no match, remove spacer and set position")
		}

		modelParams := m.Grammar.Run[data.RunIndex+1].Param
		tmpResult.Param = make([]logol.Match, len(modelParams))
		for i, param := range modelParams {
			p, ok := tmpContextVars[param]
			if !ok {
				logger.Debugf("Param %s not available adding empty one", param)
				tmpResult.Param[i] = logol.NewMatch()
				tmpResult.Param[i].Uid = m.getUID()
			} else {
				logger.Debugf("Add param %s", param)
				tmpResult.Param[i] = p
			}
		}

		tmpResult.ContextVars = make([]map[string]logol.Match, 0)
		tmpResult.RunIndex = data.RunIndex + 1
		logger.Debugf("Go to next main model %s.%s", modelTo, modelVariableTo)
		m.sendMessage(modelTo, modelVariableTo, tmpResult, false)
	}
}

func (m msgManager) checkMetaBeforeResult(model string, data logol.Result) bool {
	data.Iteration = 0
	data.Param = m.setParam(data.ContextVars[len(data.ContextVars)-1], m.Grammar.Models[model].Param)
	modelsToRun := len(m.Grammar.Run) - 1

	if len(m.Grammar.Meta) > 0 {
		logger.Debugf("Check global meta constraints")
		model := m.Grammar.Run[modelsToRun].Model
		modelsContextVars := make(map[string]logol.Match)

		// Load context vars
		for mi := 0; mi < modelsToRun; mi++ {
			currentModelParams := m.Grammar.Run[mi].Param
			for i, param := range currentModelParams {
				modelsContextVars[param] = data.ContextVars[len(data.ContextVars)-1][m.Grammar.Models[model].Param[i]]
			}

		}

		tmpMatch := logol.NewMatch()
		tmpMatch.Start = data.Matches[0].Start
		tmpMatch.End = data.Matches[len(data.Matches)-1].End
		for m := 0; m < len(data.Matches); m++ {
			tmpMatch.Sub += data.Matches[m].Sub
			tmpMatch.Indel += data.Matches[m].Indel
		}
		modelsContextVars[model] = tmpMatch
		for i := 0; i < modelsToRun; i++ {
			model := m.Grammar.Run[i].Model
			tmpMatch := logol.NewMatch()
			tmpMatch.Start = data.PrevMatches[i][0].Start
			tmpMatch.End = data.PrevMatches[i][len(data.PrevMatches[i])-1].End
			for m := 0; m < len(data.PrevMatches[i]); m++ {
				tmpMatch.Sub += data.PrevMatches[i][m].Sub
				tmpMatch.Indel += data.PrevMatches[i][m].Indel
			}
			modelsContextVars[model] = tmpMatch

		}

		for _, meta := range m.Grammar.Meta {
			logger.Debugf("Check global meta constraint %s", meta)
			isMetaOk := utils.Evaluate(meta, modelsContextVars)
			if !isMetaOk {
				logger.Debugf("Global meta check failed %s", meta)
				m.Transport.AddBan(data.Uid, 1)
				return false
			}
			logger.Debugf("Meta ok!")
		}
	}

	// data_json, _ := json.Marshal(data)
	// logger.Debugf("Match:Over:SendResult: %s", data_json)
	m.sendMessage("over", "over", data, true)
	return true
}

// Look at next variables and send result info to each of them
func (m msgManager) goNext(model string, modelVariable string, data logol.Result) {
	// Send result to next components in grammar
	nextVars := m.Grammar.Models[model].Vars[modelVariable].Next
	if len(nextVars) == 0 {
		logger.Debugf("No next var")
		logger.Debugf("First check meta constraints")
		for _, meta := range m.Grammar.Models[model].Meta {
			isMetaOk := utils.Evaluate(meta, data.ContextVars[len(data.ContextVars)-1])
			if !isMetaOk {
				m.Transport.AddBan(data.Uid, 1)
				return
			}
		}
		if len(data.From) > 0 {
			lastFrom := data.From[len(data.From)-1]
			/*
			   elts := strings.Split(lastFrom, ".")
			   backModel := elts[0]
			   backVariable := elts[1]
			*/
			backModel := lastFrom.Model
			backVariable := lastFrom.Variable
			data.CallbackUid = lastFrom.Uid
			data.From = data.From[:len(data.From)-1]
			data.Step = transport.STEP_POST
			data.Param = m.setParam(data.ContextVars[len(data.ContextVars)-1], m.Grammar.Models[model].Param)
			logger.Debugf("Go back to calling model %s %s", backModel, backVariable)
			m.sendMessage(backModel, backVariable, data, false)
		} else {
			data.Step = transport.STEP_YETTOBEDEFINED
			publishMsg := m.prepareMessage(model, modelVariable, data)
			m.publishMessage("logol-analyse-"+m.Chuid, publishMsg)
			/*
							if m.hasNextModel(data) {
								logger.Debugf("Has other main models, go to next model")
								m.goNextModel(model, data)
								return
							}
							metaOk := m.checkMetaBeforeResult(model, data)
							if !metaOk {
								logger.Debugf("Global meta constrols failed")
				            }
			*/
		}
	} else {
		logger.Debugf("Go to next vars")
		data.Iteration = 0
		for _, nextVar := range nextVars {
			m.sendMessage(model, nextVar, data, false)
		}
	}
}

// Send message to rabbitmq
func (m msgManager) publishMessage(queue string, msg string) {
	m.Transport.PublishMessage(queue, msg)
}

// Get a unique identifier
func (m msgManager) getUID() string {
	uid := uuid.Must(uuid.NewV4())
	return uid.String()
}

// Prepare message before sending it to rabbitmq
//
// Give a unique id to message, store result in redis and send uid to rabbitmq
func (m msgManager) prepareMessage(model string, modelVariable string, data logol.Result) (publishMsg string) {
	sort.Slice(data.Matches, func(i, j int) bool {
		return data.Matches[i].Start < data.Matches[j].Start
	})
	data.MsgTo = "logol-" + model + "-" + modelVariable
	data.Model = model
	data.ModelVariable = modelVariable

	publishMsg = m.Transport.PrepareMessage(data)
	return publishMsg
}

// Send current result to specified component or to result queue if over is true (meaning a full match)
//
// if some components are not yet defined, then try to define them now and go to result at least
func (m msgManager) sendMessage(model string, modelVariable string, data logol.Result, over bool) {
	// If over or final check step
	if data.Step != transport.STEP_YETTOBEDEFINED {
		m.Transport.IncrFlowStat(data.Uid, data.Model+"."+data.ModelVariable, model+"."+modelVariable)
		m.SendStat(data.Model + "." + data.ModelVariable)
	}
	if over || data.Step == transport.STEP_YETTOBEDEFINED {
		if len(data.YetToBeDefined) > 0 {
			logger.Debugf("Some vars are still pending to be analysed, should check them now")
			data.Step = transport.STEP_YETTOBEDEFINED
			publishMsg := m.prepareMessage(model, modelVariable, data)
			m.publishMessage("logol-analyse-"+m.Chuid, publishMsg)
			return
		}
		if over {
			metaOk := m.checkMetaBeforeResult(model, data)
			if !metaOk {
				logger.Debugf("Global meta constrols failed")
				return
			}
		}

		if data.ExpectNoMatch {
			begin := data.Matches[0].Start
			end := data.Matches[len(data.Matches)-1].End
			logger.Debugf("Expecting no match, compare if equals")
			if data.ExpectNoMatchVar.Start == begin && data.ExpectNoMatchVar.End == end {
				logger.Debugf("Got a match while expecting no match")
				m.Transport.AddBan(data.Uid, 1)
				m.Transport.AddToBan(data.Uid, data.ExpectNoMatchVar.Uid)
				return
			}
		}
		publishMsg := m.prepareMessage(model, modelVariable, data)
		m.publishMessage("logol-result-"+m.Chuid, publishMsg)

	} else {
		publishMsg := m.prepareMessage(model, modelVariable, data)
		m.publishMessage("logol-analyse-"+m.Chuid, publishMsg)

	}
	logger.Debugf("Sent message to %s.%s", model, modelVariable)

}

func (m msgManager) callModel(model string, modelVariable string, data logol.Result, contextVars map[string]logol.Match) {
	curVariable := m.Grammar.Models[model].Vars[modelVariable]
	callModel := curVariable.Model.Name
	tmpResult := logol.NewResult()
	tmpResult.Uid = data.Uid
	tmpResult.Outfile = data.Outfile
	tmpResult.Step = transport.STEP_PRE
	tmpResult.Iteration = data.Iteration + 1
	tmpResult.From = data.From
	data.From = make([]logol.From, 0)
	newFrom := logol.From{}
	newFrom.Model = model
	newFrom.Variable = modelVariable
	newFrom.Uid = m.getUID()
	tmpResult.From = append(tmpResult.From, newFrom)
	//tmpResult.From = append(tmpResult.From, model + "." + modelVariable)
	tmpResult.ContextVars = data.ContextVars
	tmpResult.Context = append(tmpResult.Context, data.Matches)
	tmpResult.Matches = make([]logol.Match, 0)
	tmpResult.Spacer = data.Spacer
	tmpResult.Position = data.Position
	tmpResult.Param = make([]logol.Match, 0)
	tmpResult.YetToBeDefined = data.YetToBeDefined
	tmpResult.Overlap = data.Overlap
	tmpResult.ExpectNoMatch = data.ExpectNoMatch
	tmpResult.ExpectNoMatchUID = data.ExpectNoMatchUID
	tmpResult.ExpectNoMatchVar = data.ExpectNoMatchVar

	if len(curVariable.Model.Param) > 0 {
		tmpResult.Param = m.setParam(data.ContextVars[len(data.ContextVars)-1], curVariable.Model.Param)
	}

	if curVariable.Overlap {
		tmpResult.Overlap = true
	}
	modelVariablesTo := m.Grammar.Models[callModel].Start

	for i := 0; i < len(modelVariablesTo); i++ {
		if i > 0 {
			m.Transport.AddCount(tmpResult.Uid, 1)
			//m.Client.Incr("logol:" + tmpResult.Uid + ":count")
		}
		modelVariableTo := modelVariablesTo[i]
		logger.Debugf("Call model %s:%s", callModel, modelVariableTo)
		m.sendMessage(callModel, modelVariableTo, tmpResult, false)
		if data.Step != transport.STEP_YETTOBEDEFINED {
			m.Transport.IncrFlowStat(data.Uid, data.Model+"."+data.ModelVariable, callModel+"."+modelVariableTo)
			m.SendStat(data.Model + "." + data.ModelVariable)
		}
	}

}

func (m msgManager) handleYetToBeDefined(result logol.Result, model string, modelVariable string) {
	logger.Debugf("handleYetToBeDefined for %s", model)
	index := result.GetFirstMatchAnalysable()
	if index == -1 {
		logger.Debugf("No variable in YetToBeDefined can be analysed")
		if m.hasNextModel(result) {
			m.goNextModel(model, result)
		} else {
			m.Transport.AddBan(result.Uid, 1)
		}
		//m.Client.Incr("logol:" + result.Uid + ":ban")
		return
	}
	if index == -2 {
		logger.Debugf("All yet to be defined done, sending result")
		// TODO check if a "not" model and compare with variable length
		if m.hasNextModel(result) {
			m.goNextModel(model, result)
		} else {
			if result.ExpectNoMatch {
				begin := result.Matches[0].Start
				end := result.Matches[len(result.Matches)-1].End
				logger.Debugf("Got a match while expecting no match, check if equals")
				if result.ExpectNoMatchVar.Start == begin && result.ExpectNoMatchVar.End == end {
					logger.Debugf("Got a match while expecting no match")
					m.Transport.AddBan(result.Uid, 1)
					m.Transport.AddToBan(result.Uid, result.ExpectNoMatchVar.Uid)
					return
				}
			}
			if result.ExpectNoMatchUID != "" {
				result.ExpectNoMatchUID = ""
				result.ExpectNoMatch = false
			}
			publishMsg := m.prepareMessage("over", "over", result)
			m.publishMessage("logol-result-"+m.Chuid, publishMsg)
		}
		return
	}

	matchToAnalyse := result.YetToBeDefined[index]
	minPos, preSpacer, maxPos, postSpacer := result.FindSurroundingPositions(matchToAnalyse.Uid)
	logger.Debugf("YTBD: %d, %t, %d, %t", minPos, preSpacer, maxPos, postSpacer)

	curVariable := m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id]
	saveVariable := m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id]

	if preSpacer {
		if curVariable.String_constraints.Start.Min == "" && minPos > -1 {
			curVariable.String_constraints.Start.Min = strconv.Itoa(minPos)
		}
		if curVariable.String_constraints.Start.Max == "" && maxPos > -1 {
			curVariable.String_constraints.Start.Max = strconv.Itoa(maxPos)
		}
	} else {
		if matchToAnalyse.MinPosition < minPos {
			matchToAnalyse.MinPosition = minPos
		}
		if curVariable.String_constraints.Start.Min == "" && minPos > -1 {
			curVariable.String_constraints.Start.Min = strconv.Itoa(minPos)
		}
		if curVariable.String_constraints.Start.Max == "" && minPos > -1 {
			curVariable.String_constraints.Start.Max = strconv.Itoa(minPos)
		}
	}
	if postSpacer {
		if curVariable.String_constraints.End.Min == "" && minPos > -1 {
			curVariable.String_constraints.End.Min = strconv.Itoa(minPos)
		}
		if curVariable.String_constraints.End.Max == "" && maxPos > -1 {
			curVariable.String_constraints.End.Max = strconv.Itoa(maxPos)
		}
	} else {
		if curVariable.String_constraints.End.Min == "" && maxPos > -1 {
			curVariable.String_constraints.End.Min = strconv.Itoa(maxPos)
		}
		if curVariable.String_constraints.End.Max == "" && maxPos > -1 {
			curVariable.String_constraints.End.Max = strconv.Itoa(maxPos)
		}
	}

	if preSpacer {
		matchToAnalyse.Spacer = true
	} else {
		matchToAnalyse.Spacer = false
	}

	contextVars := make(map[string]logol.Match)
	for _, uid := range matchToAnalyse.YetToBeDefined {
		for _, m := range result.Matches {
			elt, found := m.GetByUid(uid)
			if found {
				contextVars[elt.SavedAs] = elt
				break
			}
		}
	}

	if curVariable.String_constraints.Start.Min != "" && curVariable.String_constraints.Start.Min == curVariable.String_constraints.Start.Max {
		// we have a fixed position to start
		matchToAnalyse.MinPosition, _ = utils.GetRangeValue(curVariable.String_constraints.Start.Min, contextVars)
		matchToAnalyse.Spacer = false
	} else {
		if curVariable.String_constraints.End.Min != "" && curVariable.String_constraints.End.Min == curVariable.String_constraints.End.Max {
			// we have a fixed end position
			endPos, _ := utils.GetRangeValue(curVariable.String_constraints.End.Min, contextVars)
			if curVariable.HasContentConstraint() {
				content, isFixed, _ := curVariable.GetContentConstraint()
				minContentLen := 0
				maxContentLen := 0
				if isFixed {
					minContentLen = len(content)
					maxContentLen = len(content)
				} else {
					constrainedVar := contextVars[content]
					minContentLen = constrainedVar.End - constrainedVar.Start
					maxContentLen = minContentLen
				}
				if curVariable.HasDistanceConstraint() {
					_, maxDist := curVariable.GetDistanceConstraint()
					// minDistVal, _ := utils.GetRangeValue(minDist, contextVars)
					maxDistVal, _ := utils.GetRangeValue(maxDist, contextVars)
					minContentLen = minContentLen - maxDistVal
					maxContentLen = minContentLen + maxDistVal
				}
				curVariable.String_constraints.Start.Min = strconv.Itoa(endPos - maxContentLen)
				if endPos-maxContentLen < 0 {
					curVariable.String_constraints.Start.Min = "0"
				}
				curVariable.String_constraints.Start.Max = strconv.Itoa(endPos - minContentLen)
				if endPos-minContentLen < 0 {
					curVariable.String_constraints.Start.Max = "0"
				}
				if curVariable.String_constraints.Start.Min == curVariable.String_constraints.Start.Max {
					// we have a fixed position to start
					matchToAnalyse.MinPosition, _ = utils.GetRangeValue(curVariable.String_constraints.Start.Min, contextVars)
					matchToAnalyse.Spacer = false
				}

			}
		}
	}
	//json_debug, _ := json.Marshal(curVariable)
	//logger.Debugf("YTBD constraints narrow, spacer? %t, minpos: %d  , %s", matchToAnalyse.Spacer, matchToAnalyse.MinPosition, json_debug)

	m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id] = curVariable

	if !m.IsCassie && matchToAnalyse.Spacer && !matchToAnalyse.IsModel {
		// Forward to cassie
		publishMsg := m.prepareMessage(model, modelVariable, result)
		m.publishMessage("logol-cassie-"+m.Chuid, publishMsg)
		return
	}

	matchChannel := make(chan logol.Match)
	// If is model, just look at children to compute and check constraints

	// Else find it, forward to cassie if needed
	nbMatches := 0

	isModel := m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id].Model.Name != ""
	if isModel {
		go m.SearchUtils.FixModel(matchChannel, matchToAnalyse)
	} else {
		go m.SearchUtils.FindToBeAnalysed(matchChannel, m.Grammar, matchToAnalyse, result.Matches, contextVars)
	}
	result.YetToBeDefined = append(result.YetToBeDefined[:index], result.YetToBeDefined[index+1:]...)
	for match := range matchChannel {
		match.Uid = matchToAnalyse.Uid
		nbMatches++
		m.SearchUtils.UpdateByUid(match, result.Matches)
		// update result contextVars (all array elements) with match if defined
		for cv := 0; cv < len(result.ContextVars); cv++ {
			for key, value := range result.ContextVars[cv] {
				if value.Uid == match.Uid {
					result.ContextVars[cv][key] = match
				}
			}
		}
		publishMsg := m.prepareMessage("ytbd", "ytbd", result)
		m.publishMessage("logol-analyse-"+m.Chuid, publishMsg)
	}

	m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id] = saveVariable

	if nbMatches == 0 {
		//m.Client.Incr("logol:" + result.Uid + ":ban")
		if result.ExpectNoMatch {
			logger.Debugf("Expect no match, go to next model")
			if m.hasNextModel(result) {
				m.goNextModel(model, result)
			} else {
				publishMsg := m.prepareMessage("over", "over", result)
				m.publishMessage("logol-result-"+m.Chuid, publishMsg)
			}
			return
		}
		m.Transport.AddBan(result.Uid, 1)
		return
	}
	incCount := nbMatches - 1
	//m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))
	m.Transport.AddCount(result.Uid, int64(incCount))
}

func (m msgManager) SendStat(modvar string) {
	promURL := os.Getenv("LOGOL_PROM")
	if promURL != "" {
		_, err := http.Get(promURL)
		// Process response
		if err != nil {
			logger.Debugf("Could not contact prometheus process %s", promURL)
		}
	}
}

// check if all match possibilities were analysed for ExpectNoMatch cases
func (m msgManager) isAllMatchesAnalsed(result logol.Result) {
	// TODO
	if !result.ExpectNoMatch {
		return
	}

}

func (m msgManager) handleMessage(result logol.Result) {
	// Take result message and search matching data for specified model and var
	model := result.Model
	modelVariable := result.ModelVariable
	// var newContextVars map[string]logol.Match
	newContextVars := make(map[string]logol.Match)
	logger.Debugf("Received message for step %d", result.Step)
	if result.Step == transport.STEP_YETTOBEDEFINED {
		m.handleYetToBeDefined(result, model, modelVariable)
		return
	}

	if result.Step != transport.STEP_CASSIE {
		isStartModel := false
		for _, start := range m.Grammar.Models[model].Start {
			if modelVariable == start {
				isStartModel = true
				break
			}
		}
		if isStartModel {
			if len(m.Grammar.Models[model].Param) > 0 {
				for i := range m.Grammar.Models[model].Param {
					inputID := m.Grammar.Models[model].Param[i]
					if i >= len(result.Param) {
						logger.Debugf("Param not defined")
						match := logol.NewMatch()
						match.Id = inputID
						match.Model = model
						newContextVars[inputID] = match
					} else {
						newContextVars[inputID] = result.Param[i]
					}
				}
			}
			result.ContextVars = append(result.ContextVars, newContextVars)
			result.Param = make([]logol.Match, 0)
		}
	}

	contextVars := result.ContextVars[len(result.ContextVars)-1]

	if result.Step == transport.STEP_POST {
		logger.Debugf("ModelCallback:%s:%s", model, modelVariable)
		prevContext := result.Context[len(result.Context)-1]
		result.Context = result.Context[:len(result.Context)-1]
		result.ContextVars = result.ContextVars[:len(result.ContextVars)-1]
		if len(m.Grammar.Models[model].Param) > 0 {
			for i, param := range m.Grammar.Models[model].Vars[modelVariable].Model.Param {
				outputID := param
				if i < len(result.Param) {
					contextVars[outputID] = result.Param[i]
				} else {
					logger.Debugf("Param not defined %s", outputID)
					match := logol.NewMatch()
					match.Uid = m.getUID()
					match.Id = outputID
					match.Model = model
					contextVars[outputID] = match
				}

			}
		}

		result.Param = make([]logol.Match, 0)
		match := logol.NewMatch()
		match.Model = model
		match.Id = modelVariable
		match.Uid = m.getUID()
		if result.CallbackUid != "" {
			match.Uid = result.CallbackUid
		}
		match.IsModel = true
		logger.Debugf("Create var from model matches")

		for i, m := range result.Matches {
			if i == 0 {
				match.Spacer = m.Spacer
				match.MinPosition = m.MinPosition
			}
			logger.Debugf("Compare %d <? %d", match.Start, m.Start)
			if match.Start == -1 || m.Start < match.Start {
				match.Start = m.Start
			}
			if match.End == -1 || m.End > match.End {
				match.End = m.End
			}
			match.Sub += m.Sub
			match.Indel += m.Indel
			if m.Start == -1 || m.End == -1 {
				match.YetToBeDefined = append(match.YetToBeDefined, m.Uid)
			}

		}
		match, err := m.SearchUtils.PostControl(match, m.Grammar, contextVars)
		if !err {
			// TODO if *Not*, should ban match and record CallbackUid (set in nmatch.Uid) with match pos/len
			// On final results, parse matches with CallbackUid, if same start/end remove them

			logger.Debugf("New model match pos: %d, %d", match.Start, match.End)
			match.Children = result.Matches

			result.Matches = prevContext

			result.Matches = append(result.Matches, match)
			result.Step = transport.STEP_NONE
			result.Position = match.End
			result.Spacer = false
			if len(match.YetToBeDefined) > 0 {
				result.YetToBeDefined = append(result.YetToBeDefined, match)
			}

			if result.Iteration < m.Grammar.Models[model].Vars[modelVariable].Model.RepeatMax {
				logger.Debugf("Continue iteration for %s, %s", model, modelVariable)
				m.Transport.AddCount(result.Uid, 1)
				//m.Client.IncrBy("logol:" + result.Uid + ":count", 1)
				if m.Grammar.Models[model].Vars[modelVariable].Model.RepeatSpacer {
					result.Spacer = true
				}
				if m.Grammar.Models[model].Vars[modelVariable].Model.RepeatOverlap {
					result.Overlap = true
				}
				m.callModel(model, modelVariable, result, result.ContextVars[len(result.ContextVars)-1])
			}
			if result.Iteration >= m.Grammar.Models[model].Vars[modelVariable].Model.RepeatMin {
				m.goNext(model, modelVariable, result)
			} else {
				m.Transport.AddBan(result.Uid, 1)
			}
		} else {
			m.Transport.AddBan(result.Uid, 1)
			//m.Client.Incr("logol:" + result.Uid + ":ban")
		}

	} else {
		match := logol.NewMatch()
		curVariable := m.Grammar.Models[model].Vars[modelVariable]
		if curVariable.Model.Name != "" {
			logger.Debugf("Call a model")
			m.callModel(model, modelVariable, result, contextVars)
			return
		}
		if curVariable.HasReverseConstraint() {
			match.Reverse = true
		}
		match.Overlap = curVariable.Overlap
		match.Spacer = result.Spacer
		if result.Overlap {
			match.Overlap = true
		}

		if !result.Spacer && curVariable.Overlap && !result.Overlap {
			result.Overlap = true
			lastMatch := result.Matches[len(result.Matches)-1]
			newPos := result.Position - int(m.Grammar.Options["MAX_PATTERN_LENGTH"]) + 1
			if lastMatch.Start > -1 && lastMatch.End > -1 {
				newPos = result.Position - (lastMatch.End - lastMatch.Start) + 1
			}
			logger.Debugf("Overlap from %d to %d", newPos, result.Position)
			maxToPos := result.Position
			for i := newPos + 1; i <= maxToPos; i++ {
				m.Transport.AddCount(result.Uid, 1)
			}
			for i := newPos; i <= maxToPos; i++ {
				logger.Debugf("Overlap resend to pos %d", i)
				result.Position = i
				publishMsg := m.prepareMessage(model, modelVariable, result)
				m.publishMessage("logol-analyse-"+m.Chuid, publishMsg)
			}
			return
		}

		match.MinPosition = result.Position

		matchChannel := make(chan logol.Match)

		canFindMatch := true
		if !m.SearchUtils.CanFind(m.Grammar, &match, model, modelVariable, contextVars) {
			canFindMatch = false
			go m.SearchUtils.FindFuture(matchChannel, match, model, modelVariable)
		} else {
			if result.Step == transport.STEP_CASSIE {
				logger.Debugf("DEBUG in cassie")
				go m.SearchUtils.FindCassie(matchChannel, m.Grammar, match, model, modelVariable, contextVars, result.Spacer)
				result.Step = transport.STEP_NONE
			} else {
				go m.SearchUtils.Find(matchChannel, m.Grammar, match, model, modelVariable, contextVars, result.Spacer)
			}
		}
		nextVars := curVariable.Next
		nbNext := 0

		if len(nextVars) > 0 {
			nbNext = len(nextVars)
		}

		prevMatches := result.Matches
		prevFrom := result.From
		prevYetToBeDefined := result.YetToBeDefined

		nbMatches := 0
		toForward := false

		result.Spacer = false

		//for _,match := range matches {
		for match := range matchChannel {
			// Fake match to indicate that match should be forwarded to cassie queue, doing nothing here
			logger.Debugf("Got var %s", match.Id)
			if match.Id == "" {
				toForward = true
				logger.Debugf("Forward to cassie")
				continue
			}
			nbMatches++
			result.Overlap = false
			result.From = make([]logol.From, 0)
			for _, from := range prevFrom {
				result.From = append(result.From, from)
			}
			match.Uid = m.getUID()
			match.MinPosition = result.Position
			match.Spacer = result.Spacer
			result.Position = match.End

			// json_msg, _ := json.Marshal(curVariable)
			// logger.Debugf("curVariable:%s", json_msg)
			// json_match, _ := json.Marshal(match)
			// logger.Debugf("match:%s", json_match)
			if curVariable.String_constraints.SaveAs != "" {
				saveAs := curVariable.String_constraints.SaveAs
				contextVar, contextVarAlreadyDefined := contextVars[saveAs]
				logger.Debugf("SAVE AS %s %v", saveAs, contextVar)
				if contextVarAlreadyDefined {
					if contextVar.Uid != "" {
						match.Uid = contextVar.Uid
					}
				}
				contextVars[saveAs] = match
				// json_msg, _ = json.Marshal(contextVars)
				// logger.Debugf("SaveAs:%s", json_msg)
				match.SavedAs = saveAs
			}
			if !canFindMatch {
				match.From = result.From
				result.Spacer = true
				result.YetToBeDefined = append(prevYetToBeDefined, match)

			}

			// Spacer variables are not recorded, only sets spacer again
			if !match.SpacerVar {
				result.Matches = append(prevMatches, match)
			} else {
				result.Matches = append(prevMatches, match)
				result.Spacer = true
			}
			// Now disable overlap
			result.Overlap = false

			m.goNext(model, modelVariable, result)
		}

		if toForward {
			result.Step = transport.STEP_CASSIE
			publishMsg := m.prepareMessage(model, modelVariable, result)
			m.publishMessage("logol-cassie-"+m.Chuid, publishMsg)
			return
		}
		logger.Debugf("Number matches: %d, nomatch? %t", nbMatches, result.ExpectNoMatch)
		if nbMatches == 0 {
			if result.ExpectNoMatch {
				logger.Debugf("Expect no match, go to next model")
				if m.hasNextModel(result) {
					m.goNextModel(model, result)
				} else {
					publishMsg := m.prepareMessage("over", "over", result)
					m.publishMessage("logol-result-"+m.Chuid, publishMsg)
				}
				return
			}
			m.Transport.AddBan(result.Uid, 1)
			//m.Client.Incr("logol:" + result.Uid + ":ban")
			return
		}
		if nbNext > 0 {
			incCount := (nbNext * nbMatches) - 1
			//m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))
			m.Transport.AddCount(result.Uid, int64(incCount))
		} else {
			incCount := nbMatches - 1
			//m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))
			m.Transport.AddCount(result.Uid, int64(incCount))
		}

	}

	logger.Debugf("Done")
}
