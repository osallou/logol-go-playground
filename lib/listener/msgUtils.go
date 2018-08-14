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

type msgManager struct {
    Client *redis.Client
    Ch *amqp.Channel
    Chuid string
    Grammar logol.Grammar
}

func NewMsgManager(host string, ch *amqp.Channel, chuid string) msgManager {
    manager := msgManager{}
    manager.Client = newRedisClient(host)
    manager.Ch = ch
    manager.Chuid = chuid
    return manager
}

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


func (m msgManager) setParam(contextVars map[string]logol.Match, param []string) ([]logol.Match){
    // check for input/output params of model, and set them with current context variables
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

func (m msgManager) sendMessage(model string, modelVariable string, data logol.Result, over bool) {
    // Send current result to specified component or to result queue if over is true (meaning a full match)
    u1 := uuid.Must(uuid.NewV4())
    sort.Slice(data.Matches, func(i, j int) bool {
        return data.Matches[i].Start < data.Matches[j].Start
    })
    publish_msg := amqp.Publishing{}
    publish_msg.Body = []byte(u1.String())

    data.MsgTo = "logol-" + model + "-" + modelVariable
    data.Model = model
    data.ModelVariable = modelVariable

    json_msg, _ := json.Marshal(data)
    err := m.Client.Set(u1.String(), json_msg, 0).Err()
    if err != nil{
        failOnError(err, "Failed to store message")
    }


    if over {
        m.Ch.Publish(
            "", // exchange
            "logol-result-" + m.Chuid, // key
            false, // mandatory
            false, // immediate
            publish_msg,
        )
    } else {
        m.Ch.Publish(
            "", // exchange
            "logol-analyse-" + m.Chuid, // key
            false, // mandatory
            false, // immediate
            publish_msg,
        )
    }
    log.Printf("Sent message to %s", data.MsgTo)

}


func (m msgManager) call_model(model string, modelVariable string, data logol.Result, contextVars map[string]logol.Match) {
    // TODO
    curVariable := m.Grammar.Models[model].Vars[modelVariable]
    callModel := curVariable.Model.Name
    tmpResult := logol.NewResult()
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
    if len(curVariable.Model.Param) > 0 {
        tmpResult.Param = m.setParam(data.ContextVars[len(data.ContextVars) - 1], curVariable.Model.Param)
    }
    log.Printf("Call model %s:%s", callModel, m.Grammar.Models[callModel].Start)
    m.sendMessage(callModel, m.Grammar.Models[callModel].Start, tmpResult, false)

}

func (m msgManager) handleMessage(result logol.Result) {
    // Take result message and search matching data for specified model and var
    model := result.Model
    modelVariable := result.ModelVariable
    // var newContextVars map[string]logol.Match
    newContextVars := make(map[string]logol.Match)

    if modelVariable == m.Grammar.Models[model].Start {
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

        log.Printf("Create var from model matches")
        for _, m := range result.Matches {
            log.Printf("Compare %d <? %d", match.Start, m.Start)
            if (match.Start == -1 || m.Start < match.Start) {
                match.Start = m.Start
            }
            if (match.End == -1 || m.End < match.End) {
                match.End = m.End
            }
            match.Sub += m.Sub
            match.Indel += m.Indel
        }
        log.Printf("New model match pos: %d, %d", match.Start, match.End)
        match.Children = result.Matches

        result.Matches = prev_context
        result.Matches = append(result.Matches, match)
        result.Step = STEP_NONE
        result.Position = match.End

        if result.Iteration < m.Grammar.Models[model].Vars[modelVariable].Model.RepeatMax {
            log.Printf("Continue iteration for %s, %s", model, modelVariable)
            m.Client.IncrBy("logol:count", 1)
            m.call_model(model, modelVariable, result, result.ContextVars[len(result.ContextVars) - 1])
        }
        m.go_next(model, modelVariable, result)

    } else {
        match := logol.NewMatch()
        curVariable := m.Grammar.Models[model].Vars[modelVariable]
        if curVariable.Model.Name != "" {
            log.Printf("Call a model")
            m.call_model(model, modelVariable, result, contextVars)
            return
        }

        match.MinPosition = result.Position

        matches := seq.Find(m.Grammar, match, model, modelVariable, contextVars, result.Spacer)
        nextVars := curVariable.Next
        nbNext := 0

        if len(matches) == 0 {
            m.Client.Incr("logol:ban")
            return
        }
        if len(nextVars) > 0 {
            nbNext = len(nextVars)
        }
        if nbNext > 0 {
            incCount := (nbNext * len(matches)) - 1
            m.Client.IncrBy("logol:count", int64(incCount))
        }else {
            incCount := len(matches) - 1
            m.Client.IncrBy("logol:count", int64(incCount))
        }

        prevMatches := result.Matches
        prevFrom := result.From

        result.Spacer = false

        for _,match := range matches {
            result.From = make([]string, 0)
            for _, from := range prevFrom {
                result.From = append(result.From, from)
            }
            result.Position = match.End
            result.Matches = append(prevMatches, match)
            json_msg, _ := json.Marshal(curVariable)
            log.Printf("curVariable:%s", json_msg)
            if curVariable.String_constraints.SaveAs != "" {
                //TODO
                save_as := curVariable.String_constraints.SaveAs
                contextVars[save_as] = match
                json_msg, _ = json.Marshal(contextVars)
                log.Printf("SaveAs:%s", json_msg)
            }
            m.go_next(model, modelVariable, result)
        }

    }

    log.Printf("Done")
}


func sendStats(model string, variable string, duration int64){
    // TODO
}
