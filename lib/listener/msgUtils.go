// Manage input messages

package logol


import (
    "encoding/json"
    "fmt"
    "log"
    "sort"
    "strings"
    logol "org.irisa.genouest/logol/lib/types"
    seq "org.irisa.genouest/logol/lib/sequence"
    redis "github.com/go-redis/redis"
    "github.com/streadway/amqp"
    "github.com/satori/go.uuid"
)

// Initialize a connection to redis
func newRedisClient(host string) (client *redis.Client){
	redisClient := redis.NewClient(&redis.Options{
		Addr:     host + ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := redisClient.Ping().Result()
	fmt.Println(pong, err)
	return redisClient
}

// Structure managing global access to different tools and information
type msgManager struct {
    Client *redis.Client
    Ch *amqp.Channel
    Chuid string
    Grammar logol.Grammar
    CassieManager logol.Cassie
    SearchUtils seq.SearchUtils
}

func NewMsgManager(host string, ch *amqp.Channel, chuid string) msgManager {
    manager := msgManager{}
    manager.Client = newRedisClient(host)
    manager.Ch = ch
    manager.Chuid = chuid
    return manager
}

func (m msgManager) SetSearchUtils(sequencePath string) (seq.SearchUtils){
    //m.SearchUtils = seq.NewSearchUtils(sequencePath)
    return seq.NewSearchUtils(sequencePath)
}

// Get from redis message value from input uid
func (m msgManager) get(uid string) (result logol.Result, err error) {
    // fetch from redis the message based on provided uid
    // Once fetched, delete it from db
    val, err := m.Client.Get(uid).Result()
    if err == redis.Nil {
        return logol.Result{}, err
    }
    result = logol.Result{}
    json.Unmarshal([]byte(val), &result)
    m.Client.Del(uid)
    return result, err
}


// check for input/output params of model, and set them with current context variables
func (m msgManager) setParam(contextVars map[string]logol.Match, param []string) ([]logol.Match){
    var res []logol.Match
    for _, modelOutput := range param {
        log.Printf("Set param %s", modelOutput)
        cv, ok := contextVars[modelOutput]
        if ! ok {
            // DEBUG json print for debug
            json_vars, _ := json.Marshal(contextVars)
            log.Printf("Param not in contextVars: %s", json_vars)
            m := logol.NewMatch()
            m.Id = modelOutput
            res = append(res, m)
        }else {
            log.Printf("Param in contextVars")
            res = append(res, cv)
        }
    }
    return res
}


// Look at next variables and send result info to each of them
func (m msgManager) go_next(model string, modelVariable string, data logol.Result){
    // Send result to next components in grammar
    nextVars := m.Grammar.Models[model].Vars[modelVariable].Next
    if len(nextVars) == 0 {
        log.Printf("No next var")
        if len(data.From) > 0 {
            lastFrom := data.From[len(data.From) - 1]
            elts := strings.Split(lastFrom, ".")
            backModel := elts[0]
            backVariable := elts[1]
            data.From = data.From[:len(data.From) - 1]
            data.Step = STEP_POST
            data.Param = m.setParam(data.ContextVars[len(data.ContextVars) - 1], m.Grammar.Models[model].Param)
            log.Printf("Go back to calling model %s %s", backModel, backVariable)
            m.sendMessage(backModel, backVariable, data, false)
        }else {
            modelsToRun := len(m.Grammar.Run) - 1
            log.Printf("Other main models? %d vs %d", data.RunIndex, modelsToRun)
            if data.RunIndex < modelsToRun {
                // Run next main model
                modelTo := m.Grammar.Run[data.RunIndex + 1].Model
                modelVariablesTo := m.Grammar.Models[modelTo].Start
                for i := 0; i < len(modelVariablesTo); i++ {
                    if i > 0 {
                        m.Client.Incr("logol:" + data.Uid + ":count")
                    }
                    modelVariableTo := modelVariablesTo[i]
                    log.Printf("Go to next main model %s:%s", modelTo, modelVariableTo)
                    tmpResult := logol.NewResult()
                    tmpResult.Uid = data.Uid
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
                        tmpContextVars[param] = data.ContextVars[len(data.ContextVars) - 1][m.Grammar.Models[model].Param[i]]
                    }
                    modelParams := m.Grammar.Run[data.RunIndex + 1].Param
                    tmpResult.Param = make([]logol.Match, len(modelParams))
                    for i, param := range modelParams {
                        p, ok := tmpContextVars[param]
                        if ! ok {
                            log.Printf("Param %s not available adding empty one", param)
                            tmpResult.Param[i] = logol.NewMatch()
                        }else {
                            log.Printf("Add param %s", param)
                            tmpResult.Param[i] = p
                        }
                    }
                    debug_json, _ := json.Marshal(tmpResult)
                    log.Printf("Send to main model: %s", debug_json)

                    tmpResult.ContextVars = make([]map[string]logol.Match, 0)
                    tmpResult.RunIndex = data.RunIndex + 1
                    m.sendMessage(modelTo, modelVariableTo, tmpResult, false)
                }
                return
            }

            data.Iteration = 0
            data.Param = m.setParam(data.ContextVars[len(data.ContextVars) - 1], m.Grammar.Models[model].Param)
            data_json, _ := json.Marshal(data)
            log.Printf("Match:Over:SendResult: %s", data_json)
            m.sendMessage("over", "over", data, true)
        }
    } else {
        log.Printf("Go to next vars")
        data.Iteration = 0
        for _, nextVar := range nextVars {
            m.sendMessage(model, nextVar, data, false)
        }
    }
}

// Send message to rabbitmq
func (m msgManager) publishMessage(queue string, msg amqp.Publishing){
    m.Ch.Publish(
        "", // exchange
        queue, // key
        false, // mandatory
        false, // immediate
        msg,
    )
}


// Get a unique identifier
func (m msgManager) getUid() (string) {
    uid := uuid.Must(uuid.NewV4())
    return uid.String()
}


// Prepare message before sending it to rabbitmq
//
// Give a unique id to message, store result in redis and send uid to rabbitmq
func (m msgManager) prepareMessage(model string, modelVariable string, data logol.Result) (publish_msg amqp.Publishing){
    u1 := uuid.Must(uuid.NewV4())
    sort.Slice(data.Matches, func(i, j int) bool {
        return data.Matches[i].Start < data.Matches[j].Start
    })
    publish_msg = amqp.Publishing{}
    publish_msg.Body = []byte(u1.String())

    data.MsgTo = "logol-" + model + "-" + modelVariable
    data.Model = model
    data.ModelVariable = modelVariable

    json_msg, _ := json.Marshal(data)
    err := m.Client.Set(u1.String(), json_msg, 0).Err()
    if err != nil{
        failOnError(err, "Failed to store message")
    }
    return publish_msg
}

// Send current result to specified component or to result queue if over is true (meaning a full match)
//
// if some components are not yet defined, then try to define them now and go to result at least
func (m msgManager) sendMessage(model string, modelVariable string, data logol.Result, over bool) {
    // If over or final check step
    if over || data.Step == STEP_YETTOBEDEFINED {
        if len(data.YetToBeDefined) > 0 {
            log.Printf("Some vars are still pending to be analysed, should check them now")
            data.Step = STEP_YETTOBEDEFINED
            publish_msg := m.prepareMessage(model, modelVariable, data)
            m.publishMessage("logol-analyse-" + m.Chuid, publish_msg)
            return
        } else {
            publish_msg := m.prepareMessage(model, modelVariable, data)
            m.publishMessage("logol-result-" + m.Chuid, publish_msg)
        }

    } else {
        publish_msg := m.prepareMessage(model, modelVariable, data)
        m.publishMessage("logol-analyse-" + m.Chuid, publish_msg)

    }
    log.Printf("Sent message to %s.%s", model, modelVariable)

}


func (m msgManager) call_model(model string, modelVariable string, data logol.Result, contextVars map[string]logol.Match) {
    // TODO
    curVariable := m.Grammar.Models[model].Vars[modelVariable]
    callModel := curVariable.Model.Name
    tmpResult := logol.NewResult()
    tmpResult.Uid = data.Uid
    tmpResult.Step = STEP_PRE
    tmpResult.Iteration = data.Iteration + 1
    tmpResult.From = data.From
    data.From = make([]string, 0)
    tmpResult.From = append(tmpResult.From, model + "." + modelVariable)
    tmpResult.ContextVars = data.ContextVars
    tmpResult.Context = append(tmpResult.Context, data.Matches)
    tmpResult.Matches = make([]logol.Match, 0)
    tmpResult.Spacer = data.Spacer
    tmpResult.Position = data.Position
    tmpResult.Param = make([]logol.Match, 0)
    tmpResult.YetToBeDefined = data.YetToBeDefined
    if len(curVariable.Model.Param) > 0 {
        tmpResult.Param = m.setParam(data.ContextVars[len(data.ContextVars) - 1], curVariable.Model.Param)
    }
    modelVariablesTo := m.Grammar.Models[callModel].Start
    for i := 0; i < len(modelVariablesTo); i++ {
        if i > 0 {
            m.Client.Incr("logol:" + tmpResult.Uid + ":count")
        }
        modelVariableTo := modelVariablesTo[i]
        log.Printf("Call model %s:%s", callModel, modelVariableTo)
        m.sendMessage(callModel, modelVariableTo, tmpResult, false)
    }

}

func (m msgManager) handleMessage(result logol.Result) {
    // Take result message and search matching data for specified model and var
    model := result.Model
    modelVariable := result.ModelVariable
    // var newContextVars map[string]logol.Match
    newContextVars := make(map[string]logol.Match)
    log.Printf("Received message for step %d", result.Step)
    if result.Step == STEP_YETTOBEDEFINED {
        index := result.GetFirstMatchAnalysable()
        if index == -1 {
            log.Printf("No variable in YetToBeDefined can be analysed, stopping here")
            m.Client.Incr("logol:" + result.Uid + ":ban")
            return
        }
        if index == -2 {
            log.Printf("All yet to be defined done, sending result")
            publish_msg := m.prepareMessage("over", "over", result)
            m.publishMessage("logol-result-" + m.Chuid, publish_msg)
            return
        }
        if result.YetToBeDefined[index].NeedCassie {
            // Forward to cassie
            publish_msg := m.prepareMessage(model, modelVariable, result)
            m.publishMessage("logol-cassie-" + m.Chuid, publish_msg)
            return
        }

        matchToAnalyse := result.YetToBeDefined[index]
        matchChannel := make(chan logol.Match)
        // If is model, just look at children to compute and check constraints
        // TODO manage model case

        // Else find it, forward to cassie if needed
        nbMatches := 0

        isModel := m.Grammar.Models[matchToAnalyse.Model].Vars[matchToAnalyse.Id].Model.Name != ""
        if isModel {
            go m.SearchUtils.FixModel(matchChannel, matchToAnalyse)
        } else {
            go m.SearchUtils.FindToBeAnalysed(matchChannel, m.Grammar, matchToAnalyse, result.Matches, m.CassieManager.Searcher)
        }
        result.YetToBeDefined = append(result.YetToBeDefined[:index], result.YetToBeDefined[index+1:]...)
        for match := range matchChannel {
            match.Uid = matchToAnalyse.Uid
            nbMatches += 1
            m.SearchUtils.UpdateByUid(match, result.Matches)
            publish_msg := m.prepareMessage("ytbd", "ytbd", result)
            m.publishMessage("logol-analyse-" + m.Chuid, publish_msg)
        }

        if nbMatches == 0 {
            m.Client.Incr("logol:" + result.Uid + ":ban")
            return
        }
        incCount := nbMatches - 1
        m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))


        return

    }

    if result.Step != STEP_CASSIE {
        isStartModel := false
        for _, start := range m.Grammar.Models[model].Start {
            if modelVariable == start {
                isStartModel = true
                break
            }
        }
        if isStartModel {
            if len(m.Grammar.Models[model].Param) > 0 {
                for i, _ := range m.Grammar.Models[model].Param {
                    inputId :=  m.Grammar.Models[model].Param[i]
                    if i >= len(result.Param) {
                        log.Printf("Param not defined")
                        match := logol.NewMatch()
                        match.Id = inputId
                        match.Model = model
                        newContextVars[inputId] = match
                    }else{
                        newContextVars[inputId] = result.Param[i]
                    }
                }
            }
            result.ContextVars = append(result.ContextVars, newContextVars)
            result.Param = make([]logol.Match, 0)
        }
    }

    contextVars := result.ContextVars[len(result.ContextVars) - 1]

    if result.Step == STEP_POST {
        log.Printf("ModelCallback:%s:%s", model, modelVariable)
        prev_context := result.Context[len(result.Context) - 1]
        result.Context = result.Context[:len(result.Context) - 1]
        result.ContextVars = result.ContextVars[:len(result.ContextVars) - 1]
        if len(m.Grammar.Models[model].Param) > 0 {
            for i, param := range m.Grammar.Models[model].Vars[modelVariable].Model.Param {
                outputId := param
                if i < len(result.Param) {
                    contextVars[outputId] = result.Param[i]
                }else {
                    log.Printf("Param not defined %s", outputId)
                    match := logol.NewMatch()
                    match.Id = outputId
                    match.Model = model
                    contextVars[outputId] = match
                }

            }
        }
        result.Param = make([]logol.Match, 0)
        match := logol.NewMatch()
        match.Model = model
        match.Id = modelVariable
        match.Uid = m.getUid()
        log.Printf("Create var from model matches")
        // TODO check if some childs are YetToBeDefined, if yes, mark model with YetToBeDefined
        // Sets however what can be done and add subvars to match.YetToBeDefined
        for i, m := range result.Matches {
            if i ==0 {
                match.Spacer = m.Spacer
                match.MinPosition = m.MinPosition
            }
            log.Printf("Compare %d <? %d", match.Start, m.Start)
            if (match.Start == -1 || m.Start < match.Start) {
                match.Start = m.Start
            }
            if (match.End == -1 || m.End > match.End) {
                match.End = m.End
            }
            match.Sub += m.Sub
            match.Indel += m.Indel
            if m.Start == -1 || m.End == -1 {
                match.YetToBeDefined = append(match.YetToBeDefined, m.Uid)
            }

        }
        match, err := m.SearchUtils.PostControl(match, m.Grammar, contextVars)
        if ! err {
            log.Printf("New model match pos: %d, %d", match.Start, match.End)
            match.Children = result.Matches

            result.Matches = prev_context

            result.Matches = append(result.Matches, match)
            result.Step = STEP_NONE
            result.Position = match.End
            result.Spacer = false
            if len(match.YetToBeDefined) > 0 {
                result.YetToBeDefined = append(result.YetToBeDefined, match)
            }


            if result.Iteration < m.Grammar.Models[model].Vars[modelVariable].Model.RepeatMax {
                log.Printf("Continue iteration for %s, %s", model, modelVariable)
                m.Client.IncrBy("logol:" + result.Uid + ":count", 1)
                m.call_model(model, modelVariable, result, result.ContextVars[len(result.ContextVars) - 1])
            }
            m.go_next(model, modelVariable, result)
        } else {
            m.Client.Incr("logol:" + result.Uid + ":ban")
        }

    } else {
        match := logol.NewMatch()
        curVariable := m.Grammar.Models[model].Vars[modelVariable]
        if curVariable.Model.Name != "" {
            log.Printf("Call a model")
            m.call_model(model, modelVariable, result, contextVars)
            return
        }
        match.Spacer = result.Spacer

        match.MinPosition = result.Position

        matchChannel := make(chan logol.Match)

        canFindMatch := true
        if ! m.SearchUtils.CanFind(m.Grammar, &match, model, modelVariable, contextVars) {
            canFindMatch = false
            // TODO, store in result.YetToBeDefined, add empty match with var name and model and continue
            // should check and update later on
            go m.SearchUtils.FindFuture(matchChannel, match, model, modelVariable)
        } else {
            if result.Step == STEP_CASSIE {
                log.Printf("DEBUG in cassie")
                go m.SearchUtils.FindCassie(matchChannel, m.Grammar, match, model, modelVariable, contextVars, result.Spacer, m.CassieManager.Searcher)
                result.Step = STEP_NONE
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
            log.Printf("Got var %s", match.Id)
            if match.Id == "" {
                toForward = true
                log.Printf("Forward to cassie")
                continue
            }
            nbMatches += 1
            result.From = make([]string, 0)
            for _, from := range prevFrom {
                result.From = append(result.From, from)
            }
            match.Uid = m.getUid()
            match.MinPosition = result.Position
            match.Spacer = result.Spacer
            result.Position = match.End

            json_msg, _ := json.Marshal(curVariable)
            log.Printf("curVariable:%s", json_msg)
            json_match, _ := json.Marshal(match)
            log.Printf("match:%s", json_match)
            if curVariable.String_constraints.SaveAs != "" {
                //TODO
                save_as := curVariable.String_constraints.SaveAs
                contextVar, contextVarAlreadyDefined := contextVars[save_as]
                if contextVarAlreadyDefined {
                    match.Uid = contextVar.Uid
                }
                contextVars[save_as] = match
                json_msg, _ = json.Marshal(contextVars)
                log.Printf("SaveAs:%s", json_msg)
                match.SavedAs = save_as
            }
            if ! canFindMatch {
                match.From = result.From
                result.Spacer = true
                result.YetToBeDefined = append(prevYetToBeDefined, match)

            }
            result.Matches = append(prevMatches, match)

            m.go_next(model, modelVariable, result)
        }
        if toForward {
            result.Step = STEP_CASSIE
            publish_msg := m.prepareMessage(model, modelVariable, result)
            m.publishMessage("logol-cassie-" + m.Chuid, publish_msg)
            return
        }
        if nbMatches == 0 {
            m.Client.Incr("logol:" + result.Uid + ":ban")
            return
        }
        if nbNext > 0 {
            incCount := (nbNext * nbMatches) - 1
            m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))
        }else {
            incCount := nbMatches - 1
            m.Client.IncrBy("logol:" + result.Uid + ":count", int64(incCount))
        }

    }

    log.Printf("Done")
}


func sendStats(model string, variable string, duration int64){
    // TODO
}
