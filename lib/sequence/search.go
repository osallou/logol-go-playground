package logol

import (
    "encoding/json"
    // "fmt"
    "log"
    //"regexp"
    "github.com/satori/go.uuid"
    logol "org.irisa.genouest/logol/lib/types"
    cassie "org.irisa.genouest/cassiopee"
)

func FixModel(mch chan logol.Match, match logol.Match) {
    match.Sub = 0
    match.Indel = 0
    for i, m := range match.Children {
        if i ==0 {
            match.Spacer = m.Spacer
            match.MinPosition = m.MinPosition
        }
        if (match.Start == -1 || m.Start < match.Start) {
            match.Start = m.Start
        }
        if (match.End == -1 || m.End > match.End) {
            match.End = m.End
        }
        match.Sub += m.Sub
        match.Indel += m.Indel
    }
    mch <- match
}

func FindFuture(mch chan logol.Match, match logol.Match, model string, modelVariable string) {
    tmpMatch := match.Clone()
    tmpMatch.Id = modelVariable
    tmpMatch.Model = model
    /*
    tmpMatch := logol.NewMatch()
    tmpMatch.Id = modelVariable
    tmpMatch.Model = model
    tmpMatch.YetToBeDefined = match.YetToBeDefined
    */
    mch <- tmpMatch
    close(mch)
}

func UpdateByUid(match logol.Match, matches []logol.Match){
    // find matching element in array of matches and update ones matching uid
    json_m, _ := json.Marshal(match)
    log.Printf("Update match now: %s", json_m)
    if len(matches) == 0 {
        return
    }
    log.Printf("Search uid %s in matches", match.Uid)
    for i, m := range matches {
        if m.Uid == match.Uid {
            log.Printf("Gotcha, found it")
            matches[i] = match
            break
        } else {
            UpdateByUid(match, m.Children)
        }
    }
    json_msg, _ := json.Marshal(matches)
    log.Printf("UpdateByUid %s", json_msg)
}

func FindToBeAnalysed(mch chan logol.Match, grammar logol.Grammar, match logol.Match, matches[]logol.Match, searchHandler cassie.CassieSearch) {
    contextVars := make(map[string]logol.Match)
    for _, uid := range match.YetToBeDefined {
        for _, m := range matches {
            elt, found := m.GetByUid(uid)
            if found {
                contextVars[elt.SavedAs] = elt
                break
            }
        }
    }
    if match.NeedCassie {
        FindCassie(mch, grammar, match, match.Model, match.Id, contextVars, match.Spacer, searchHandler)
    } else {
        Find(mch, grammar, match, match.Model, match.Id, contextVars, match.Spacer)
    }
}

func CanFind(grammar logol.Grammar, match *logol.Match, model string, modelVariable string, contextVars map[string]logol.Match) (can bool) {
    // TODO should manage different use cases
    // If cannot be found due to 1 variable, find all related vars and add them to match.ytbd
    log.Printf("Test if variable can be defined now")
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        content, ok := contextVars[contentConstraint]
        if ! ok {
            log.Printf("Depends on undefined variable %s", contentConstraint)
            m := logol.NewMatch()
            m.Uid = uuid.Must(uuid.NewV4()).String()
            contextVars[contentConstraint] = m
            match.YetToBeDefined = append(match.YetToBeDefined, m.Uid)
            // fmt.Printf("YetToBeDefined: %v", match.YetToBeDefined)
            return false
        }
        if (content.Start == -1 || content.End == -1) {
            log.Printf("Depends on non available variable %s", contentConstraint)
            match.YetToBeDefined = append(match.YetToBeDefined, content.Uid)
            return false
        }
    }
    if match.Spacer {
        match.NeedCassie = true
    }
    return true
}

func Find(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) (matches []logol.Match) {
    // TODO manage different search use cases

    if spacer {
        fakeMatch := logol.NewMatch()
        fakeMatch.Spacer = true
        fakeMatch.NeedCassie = true
        mch <- fakeMatch
        matches = append(matches, fakeMatch)
        close(mch)
        return matches
    }

    matches = FindExact(mch, grammar, match, model, modelVariable, contextVars, spacer)
    return matches
}


func FindCassie(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool, searchHandler cassie.CassieSearch) (matches []logol.Match) {
    log.Printf("Search in Cassie")
    // json_msg, _ := json.Marshal(contextVars)
    seq := Sequence{grammar.Sequence, 0, ""}
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        // log.Printf("TRY TO FETCH FROM CV %d", contextVars[contentConstraint].Start)
        curVariable.Value = seq.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
        if curVariable.Value == "" {
            close(mch)
            return
        }
    }

    searchHandler.Search(curVariable.Value)
    searchHandler.Sort()
    if searchHandler.GetMax_indel() > 0 {
        searchHandler.RemoveDuplicates()
    }
    smatches := cassie.GetMatchList(searchHandler)
    msize := smatches.Size()
    var i int64
    i = 0
    for i < msize {
        elem := smatches.Get(int(i))
        newMatch := logol.NewMatch()
        newMatch.Id = modelVariable
        newMatch.Model = model
        newMatch.Start = int(elem.GetPos())
        pLen := len(curVariable.Value)
        if(elem.GetIn() - elem.GetDel() != 0) {
            pLen = pLen + elem.GetIn() - elem.GetDel()
        }
        newMatch.End = int(elem.GetPos()) + pLen
        newMatch.Info = curVariable.Value
        if newMatch.Start < match.MinPosition {
            log.Printf("skip match at wrong position: %d" , newMatch.Start)
            continue
        }
        mch <- newMatch
        // log.Printf("DEBUG matches:%d %d %d", i, newMatch.Start, newMatch.End)
        i++
    }
    close(mch)
    return matches
}


func FindExact(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) (matches []logol.Match) {
    // TODO find only non overlapping, func for testing only
    seq := Sequence{grammar.Sequence, 0, ""}
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        curVariable.Value = seq.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
        if curVariable.Value == "" {
            close(mch)
            return
        }
    }

    log.Printf("Search %s at min pos %d, spacer: %t", curVariable.Value, match.MinPosition, spacer)

    findResults := make([][2]int, 0)
    sequence := seq.GetSequence()
    seqLen := len(sequence)
    patternLen := len(curVariable.Value)
    for i:=0; i < seqLen - patternLen; i++ {
        seqPart := sequence[i:i+patternLen]
        if seqPart == curVariable.Value {
            elts := [...]int{i, i+patternLen}
            findResults = append(findResults, elts)
        }
    }
    /*
    r, _ := regexp.Compile("(" + curVariable.Value + ")")
    sequence := seq.GetSequence()
    findResults := r.FindAllStringIndex(sequence, -1)
    */
    ban := 0
    for _, findResult := range findResults {
        startResult := findResult[0]
        endResult := findResult[1]
        if ! spacer {
            if startResult != match.MinPosition {
                log.Printf("skip match at wrong position: %d" , startResult)
                ban += 1
                continue
            }
        } else {
            if startResult < match.MinPosition {
                log.Printf("skip match at wrong position: %d" , startResult)
                ban += 1
                continue
            }
        }
        newMatch := logol.NewMatch()
        newMatch.Id = modelVariable
        newMatch.Model = model
        newMatch.Start = startResult
        newMatch.End = endResult
        newMatch.Info = curVariable.Value
        mch <- newMatch
        // matches = append(matches, newMatch)
        log.Printf("got match: %d, %d", newMatch.Start, newMatch.End)
    }
    log.Printf("got matches: %d", (len(findResults) - ban))
    close(mch)
    return matches
}
