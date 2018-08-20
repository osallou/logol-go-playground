//  Search functions and sequence access
package logol

import (
    "encoding/json"
    "log"
    "github.com/satori/go.uuid"
    logol "org.irisa.genouest/logol/lib/types"
    cassie "org.irisa.genouest/cassiopee"
)

// Sequence handler with methods to search and analyse a variable
type SearchUtils struct {
    SequenceHandler SequenceLru
}

// returns a new handler for sequence
func NewSearchUtils(sequencePath string) (su SearchUtils){
    su = SearchUtils{}
    s := NewSequence(sequencePath)
    log.Printf("NewSearchUtils, seq size: %d", s.Size)
    su.SequenceHandler = NewSequenceLru(s)
    return su
}

// Update a model attributes according to its children
func (s SearchUtils) FixModel(mch chan logol.Match, match logol.Match) {
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

// Creates a fake match for variables not yet defined
func (s SearchUtils) FindFuture(mch chan logol.Match, match logol.Match, model string, modelVariable string) {
    tmpMatch := match.Clone()
    tmpMatch.Id = modelVariable
    tmpMatch.Model = model
    mch <- tmpMatch
    close(mch)
}

// find matching element in array of matches and update ones matching uid
func (s SearchUtils) UpdateByUid(match logol.Match, matches []logol.Match){
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
            s.UpdateByUid(match, m.Children)
        }
    }
    json_msg, _ := json.Marshal(matches)
    log.Printf("UpdateByUid %s", json_msg)
}

// Find a variable that could not be found before (due to other constraints)
func (s SearchUtils) FindToBeAnalysed(mch chan logol.Match, grammar logol.Grammar, match logol.Match, matches[]logol.Match, searchHandler cassie.CassieSearch) {
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
    if match.Spacer {
        s.FindCassie(mch, grammar, match, match.Model, match.Id, contextVars, match.Spacer, searchHandler)
    } else {
        s.Find(mch, grammar, match, match.Model, match.Id, contextVars, match.Spacer)
    }
}

// Checks if a variable can be analysed now according to its constraints and current context
func (s SearchUtils) CanFind(grammar logol.Grammar, match *logol.Match, model string, modelVariable string, contextVars map[string]logol.Match) (can bool) {
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


// Find a variable in sequence
func (s SearchUtils) Find(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) (matches []logol.Match) {
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

    matches = s.FindExact(mch, grammar, match, model, modelVariable, contextVars, spacer)
    return matches
}


// Find a variable in sequence using external library cassiopee
func (s SearchUtils) FindCassie(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool, searchHandler cassie.CassieSearch) (matches []logol.Match) {
    log.Printf("Search in Cassie")
    // json_msg, _ := json.Marshal(contextVars)
    // seq := Sequence{grammar.Sequence, 0, ""}
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        // log.Printf("TRY TO FETCH FROM CV %d", contextVars[contentConstraint].Start)
        curVariable.Value = s.SequenceHandler.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
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
        newMatch, err := s.PostControl(newMatch, grammar, contextVars)
        if ! err {
            mch <- newMatch
        }
        // log.Printf("DEBUG matches:%d %d %d", i, newMatch.Start, newMatch.End)
        i++
    }
    close(mch)
    return matches
}

func (s SearchUtils) FindAny(mch chan logol.Match, match logol.Match, model string, modelVariable string, minSize int, maxSize int, spacer bool) {
    log.Printf("Search any string at min pos %d, spacer: %t", match.MinPosition, spacer)
    seqLen := s.SequenceHandler.Sequence.Size
    //sequence := seq.GetSequence()
    //seqLen := len(sequence)
    for l:=minSize;l<=maxSize;l ++ {
        patternLen := l
        maxSearchIndex := match.MinPosition + 1
        if spacer {
            maxSearchIndex = seqLen - patternLen
        }
        log.Printf("Loop over %d:%d", match.MinPosition , maxSearchIndex)
        for i:=match.MinPosition; i < maxSearchIndex; i++ {
            // seqPart := s.SequenceHandler.GetContent(i, i + patternLen)
            newMatch := logol.NewMatch()
            newMatch.Id = modelVariable
            newMatch.Model = model
            newMatch.Start = i
            newMatch.End = i + patternLen
            newMatch.Info = "*"
            mch <- newMatch
        }
    }
    close(mch)
}


func isExact(m1 string, m2 string) (res bool){
    res = m1 == m2
    return res
}

// Find an exact pattern in sequence
func (s SearchUtils) FindExact(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) (matches []logol.Match) {
    // seq := Sequence{grammar.Sequence, 0, ""}
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        curVariable.Value = s.SequenceHandler.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
        if curVariable.Value == "" {
            close(mch)
            return
        }
    }

    log.Printf("Search %s at min pos %d, spacer: %t", curVariable.Value, match.MinPosition, spacer)

    findResults := make([][2]int, 0)
    seqLen := s.SequenceHandler.Sequence.Size
    //sequence := seq.GetSequence()
    //seqLen := len(sequence)
    patternLen := len(curVariable.Value)
    for i:=0; i < seqLen - patternLen; i++ {
        seqPart := s.SequenceHandler.GetContent(i, i + patternLen)

        // seqPart := sequence[i:i+patternLen]
        if isExact(seqPart, curVariable.Value) {
            elts := [...]int{i, i+patternLen}
            findResults = append(findResults, elts)
        }
    }

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
        newMatch, err := s.PostControl(newMatch, grammar, contextVars)
        if ! err {
            mch <- newMatch
            // matches = append(matches, newMatch)
            log.Printf("got match: %d, %d", newMatch.Start, newMatch.End)
        }
    }
    log.Printf("got matches: %d", (len(findResults) - ban))
    close(mch)
    return matches
}


func (s SearchUtils) PostControl(match logol.Match, grammar logol.Grammar, contextVars map[string]logol.Match) (newMatch logol.Match, err bool){
    // TODO
    // check model global constraints
    // Check for negative_constraints
    newMatch = match
    return newMatch, false
}
