//  Search functions and sequence access
package logol

import (
    "encoding/json"
    "fmt"
    //"log"
    "github.com/satori/go.uuid"
    //"strconv"
    logol "org.irisa.genouest/logol/lib/types"
    cassie "github.com/osallou/cassiopee-go"
    utils "org.irisa.genouest/logol/lib/utils"
    logs "org.irisa.genouest/logol/lib/log"
)

var logger = logs.GetLogger("logol.sequence")

const MAX_MATCH_SIZE = 1000

// Sequence handler with methods to search and analyse a variable
type SearchUtils struct {
    SequenceHandler SequenceLru
}

// returns a new handler for sequence
func NewSearchUtils(sequencePath string) (su SearchUtils){
    su = SearchUtils{}
    s := NewSequence(sequencePath)
    logger.Debugf("NewSearchUtils, seq size: %d", s.Size)
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
    logger.Debugf("Update match now: %s", json_m)
    if len(matches) == 0 {
        return
    }
    logger.Debugf("Search uid %s in matches", match.Uid)
    for i, m := range matches {
        if m.Uid == match.Uid {
            logger.Debugf("Gotcha, found it")
            matches[i] = match
            break
        } else {
            s.UpdateByUid(match, m.Children)
        }
    }
    json_msg, _ := json.Marshal(matches)
    logger.Debugf("UpdateByUid %s", json_msg)
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
    // Manage string struct and negative constraints
    // If cannot be found due to 1 variable, find all related vars and add them to match.ytbd

    /*
    String constraint:
        Content string
        Size RangeConstraint
        Start RangeConstraint
        End RangeConstraint
    Struct constraint:
        Cost string // range [3,4]
        Distance string // range [10,20]
    */

    uniques := func (input []string) []string {
    	u := make([]string, 0, len(input))
    	m := make(map[string]bool)
    	for _, val := range input {
    		if _, ok := m[val]; !ok {
    			m[val] = true
    			u = append(u, val)
    		}
    	}
    	return u
    }
    logger.Debugf("Test if variable can be defined now")
    curVariable := grammar.Models[model].Vars[modelVariable]
    hasUndefined := false
    undefinedVars := make([]string, 0)
    _hasUndefined, _undefinedVars := utils.HasUndefinedRangeVars(curVariable.Value, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.Content, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.Size.Min, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.Size.Max, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.Start.Min, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.Start.Max, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.End.Min, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.String_constraints.End.Max, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }

    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.Struct_constraints.Cost.Min, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.Struct_constraints.Cost.Max, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.Struct_constraints.Distance.Min, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }
    _hasUndefined, _undefinedVars = utils.HasUndefinedRangeVars(curVariable.Struct_constraints.Distance.Max, contextVars)
    if _hasUndefined {
        hasUndefined = true
        undefinedVars = append(undefinedVars, _undefinedVars...)
    }

    undefinedVars = uniques(undefinedVars)

    nbUndefinedVars := len(undefinedVars)
    for i:=0;i<nbUndefinedVars;i++ {
            contentConstraint := undefinedVars[i]
            content, ok := contextVars[contentConstraint]
            if ! ok {
                logger.Debugf("Depends on undefined variable %s", contentConstraint)
                m := logol.NewMatch()
                m.Uid = uuid.Must(uuid.NewV4()).String()
                contextVars[contentConstraint] = m
                match.YetToBeDefined = append(match.YetToBeDefined, m.Uid)
                // fmt.Printf("YetToBeDefined: %v", match.YetToBeDefined)
                return false
            }
            if (content.Start == -1 || content.End == -1) {
                logger.Debugf("Depends on non available variable %s", contentConstraint)
                match.YetToBeDefined = append(match.YetToBeDefined, content.Uid)
                return false
            }
    }
    if hasUndefined {
        return false
    } else {
        return true
    }

}


// Find a variable in sequence
func (s SearchUtils) Find(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) {
    // TODO manage different search use cases

    curVariable := grammar.Models[model].Vars[modelVariable]
    // Spacer variable, just set spacer var and continue
    if curVariable.Value == "_" {
        fakeMatch := logol.NewMatch()
        fakeMatch.Id = modelVariable
        fakeMatch.Model = model
        fakeMatch.Spacer = true
        fakeMatch.SpacerVar = true
        fakeMatch.Start = fakeMatch.MinPosition
        fakeMatch.End = fakeMatch.MinPosition
        // fakeMatch.NeedCassie = true
        mch <- fakeMatch
        //matches = append(matches, fakeMatch)
        close(mch)
        return
    }

    if spacer {
        fakeMatch := logol.NewMatch()
        fakeMatch.Spacer = true
        // fakeMatch.NeedCassie = true
        mch <- fakeMatch
        //matches = append(matches, fakeMatch)
        close(mch)
        return
        //return matches
    }

    if curVariable.HasStartConstraint(){
        // Check current position
        minStart, maxStart := curVariable.GetStartConstraint()
        min, _ := utils.GetRangeValue(minStart, contextVars)
        max, _ := utils.GetRangeValue(maxStart, contextVars)
        if ! curVariable.Overlap {
            if (min != -1 && match.MinPosition < min) || (max != -1 && match.MinPosition > max) {
                close(mch)
                return
            }
        } else {
            logger.Debugf("Skipping start constraint for the moment as overlap is allowed")
        }
    }

    if curVariable.HasSizeConstraint() && ! curVariable.HasContentConstraint(){
        logger.Debugf("Size constraint but no content constraint, take any chars")
        min, err := utils.GetRangeValue(curVariable.String_constraints.Size.Min, contextVars)
        if err {
            logger.Debugf("Could not interpret constraint, skipping it: %s", curVariable.String_constraints.Size.Min)
            min = 1
        }
        max, err := utils.GetRangeValue(curVariable.String_constraints.Size.Max, contextVars)
        if err {
            logger.Debugf("Could not interpret constraint, skipping it: %s", curVariable.String_constraints.Size.Max)
            max = MAX_MATCH_SIZE
        }
        //min, _ := strconv.Atoi(curVariable.String_constraints.Size.Min)
        //max, _ := strconv.Atoi(curVariable.String_constraints.Size.Max)
        s.FindAny(
            mch ,
            grammar,
            match,
            model,
            modelVariable,
            min,
            max,
            contextVars,
            spacer)
    } else {
        // Has content constraint
        if curVariable.HasCostConstraint() || curVariable.HasDistanceConstraint(){
            maxCost := -1
            maxDist := -1
            if curVariable.HasCostConstraint() {
                _, smaxCost := curVariable.GetCostConstraint()
                // minCost, _ = utils.GetRangeValue(sminCost, contextVars)
                maxCost, _ = utils.GetRangeValue(smaxCost, contextVars)
            }
            if curVariable.HasDistanceConstraint() {
                _, smaxDist := curVariable.GetDistanceConstraint()
                // minDist, _ = utils.GetRangeValue(sminDist, contextVars)
                maxDist, _ = utils.GetRangeValue(smaxDist, contextVars)
            }
            s.FindApproximate(mch, grammar, match, model, modelVariable, contextVars, spacer, maxCost, maxDist)
        }else{
            s.FindExact(mch, grammar, match, model, modelVariable, contextVars, spacer)
        }
    }

}

func (s SearchUtils) FindApproximate(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool, maxCost int, maxDistance int) {
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        logger.Debugf("FindExact, get var content %s", contentConstraint)
        curVariable.Value = s.SequenceHandler.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
        logger.Debugf("? %s",curVariable.Value)
        if curVariable.Value == "" {
            close(mch)
            return
        }
    }


    findResults := make([][4]int, 0)
    seqLen := s.SequenceHandler.Sequence.Size
    patternLen := len(curVariable.Value)
    minStart := match.MinPosition
    maxStart := match.MinPosition + 1
    if match.Spacer {
        maxStart = seqLen - patternLen + 1
    }

    logger.Debugf("seach between %d and %d", minStart, maxStart)
    for i:=minStart; i < maxStart; i++ {
        seqPart := s.SequenceHandler.GetContent(i, i + patternLen + 1+ maxDistance)

        bioString := NewDnaString(curVariable.Value)
        if match.Reverse {
            curVariable.Value = bioString.Reverse()
        }
        if curVariable.HasMorphism() {
            bioString.SetMorphisms(curVariable.GetMorphism(grammar.Morphisms).Morph)
        }
        b1 := NewDnaString(curVariable.Value)
        approxResults := IsApproximate(&b1, seqPart, 0, maxCost, 0, 0, maxDistance)
        nbApproxResults := len(approxResults)
        if nbApproxResults > 0 {
            for r:=0;r<nbApproxResults;r++ {
                approxResult := approxResults[r]
                length := patternLen + approxResult[2] - approxResult[3]
                elts := [...]int{i, i+length, approxResult[1], approxResult[2] + approxResult[3]}
                findResults = append(findResults, elts)
            }
        }
    }


    uniques := func (input [][4]int) [][4]int {
        u := make([][4]int, 0, len(input))
        m := make(map[string][4]int)
        for _, val := range input {
            key := fmt.Sprintf("%d-%d-%d-%d", val[0], val[1], val[2], val[3])
            if _, ok := m[key]; !ok {
                m[key] = val
                u = append(u, val)
            }
        }
        return u
    }
    // Remove duplicates if indel allowed
    if maxDistance > 0 {
        findResults = uniques(findResults)
    }

    ban := 0
    for _, findResult := range findResults {
        startResult := findResult[0]
        endResult := findResult[1]
        if ! spacer {
            if startResult != match.MinPosition {
                logger.Debugf("skip match at wrong position: %d" , startResult)
                ban += 1
                continue
            }
        } else {
            if startResult < match.MinPosition {
                logger.Debugf("skip match at wrong position: %d" , startResult)
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
        newMatch.Sub = findResult[2]
        newMatch.Indel = findResult[3]
        newMatch, err := s.PostControl(newMatch, grammar, contextVars)
        if ! err {
            mch <- newMatch
            // matches = append(matches, newMatch)
            logger.Debugf("got match: %d, %d", newMatch.Start, newMatch.End)
        }
    }
    logger.Debugf("got matches: %d", (len(findResults) - ban))

    close(mch)
}

// Find a variable in sequence using external library cassiopee
func (s SearchUtils) FindCassie(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool, searchHandler cassie.CassieSearch) {
    logger.Debugf("Search in Cassie")
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
    maxCost := -1
    maxDist := -1
    if curVariable.HasCostConstraint() {
        _, smaxCost := curVariable.GetCostConstraint()
        maxCost, _ = utils.GetRangeValue(smaxCost, contextVars)
    }
    if curVariable.HasDistanceConstraint() {
        _, smaxDist := curVariable.GetDistanceConstraint()
        maxDist, _ = utils.GetRangeValue(smaxDist, contextVars)
    }

    searchHandler.SetAmbiguity(true)

    if maxDist > 0 {
        searchHandler.SetMax_indel(maxDist)
    }
    if maxCost > 0 {
        searchHandler.SetMax_subst(maxCost)
    }

    bioString := NewDnaString(curVariable.Value)
    if match.Reverse {
        curVariable.Value = bioString.Reverse()
    }
    // TODO set morphism support in cassiopee
    if curVariable.HasMorphism() {
        logger.Debugf("Use morphisms with cassie")
        searchMorph := make(map[string]string)
        morph := curVariable.GetMorphism(grammar.Morphisms)
        //json_msg, _ := json.Marshal(morph)
        //log.Printf("Morph content %s", json_msg)
        smap := cassie.NewMapStringString()
        for key, value := range morph.Morph {

            svalue := ""
            for _, sval := range value {
                svalue += sval
            }
            searchMorph[key] = svalue

            smap.Set(key, svalue)
        }
        //smap := cassie.NewMapStringString(searchMorph)
        searchHandler.SetMorphisms(smap)

    }else {
        smap := cassie.NewMapStringString()
        searchHandler.SetMorphisms(smap)
    }

    searchHandler.Search(curVariable.Value)
    searchHandler.Sort()
    searchHandler.RemoveDuplicates()
    /*
    if searchHandler.GetMax_indel() > 0 {
        searchHandler.RemoveDuplicates()
    }*/
    smatches := cassie.GetMatchList(searchHandler)
    msize := smatches.Size()
    var i int64
    i = 0
    logger.Debugf("Cassie found %d solutions", msize)
    for i < msize {
        elem := smatches.Get(int(i))
        newMatch := logol.NewMatch()
        newMatch.Id = modelVariable
        newMatch.Model = model
        newMatch.Start = int(elem.GetPos())
        newMatch.Sub = elem.GetSubst()
        newMatch.Indel = elem.GetIn() + elem.GetDel()
        pLen := len(curVariable.Value)
        if(elem.GetIn() - elem.GetDel() != 0) {
            pLen = pLen + elem.GetIn() - elem.GetDel()
        }
        newMatch.End = int(elem.GetPos()) + pLen
        newMatch.Info = curVariable.Value
        logger.Debugf("Cassie found %d:%d:%d:%d", newMatch.Start, newMatch.End, newMatch.Sub, newMatch.Indel)
        if newMatch.Start < match.MinPosition {
            logger.Debugf("skip match at wrong position: %d" , newMatch.Start)
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
}

func (s SearchUtils) FindAny(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, minSize int, maxSize int, contextVars map[string]logol.Match, spacer bool) {
    logger.Debugf("Search any string at min pos %d, spacer: %t", match.MinPosition, spacer)
    seqLen := s.SequenceHandler.Sequence.Size
    //sequence := seq.GetSequence()
    //seqLen := len(sequence)
    logger.Debugf("Extract string of size %d -> %d", minSize, maxSize)
    for l:=minSize;l<=maxSize;l ++ {
        patternLen := l
        maxSearchIndex := match.MinPosition + 1
        if spacer {
            maxSearchIndex = seqLen - patternLen
        }
        logger.Debugf("Loop over %d:%d", match.MinPosition , maxSearchIndex)
        for i:=match.MinPosition; i < maxSearchIndex; i++ {
            // seqPart := s.SequenceHandler.GetContent(i, i + patternLen)
            newMatch := logol.NewMatch()
            newMatch.Id = modelVariable
            newMatch.Model = model
            newMatch.Start = i
            newMatch.End = i + patternLen
            newMatch.Info = "*"
            newMatch, err := s.PostControl(newMatch, grammar, contextVars)
            if ! err {
                mch <- newMatch
                // matches = append(matches, newMatch)
                logger.Debugf("got match: %d, %d", newMatch.Start, newMatch.End)
            }
        }
    }
    close(mch)
}

// compare pattern vs sequence
func isExact(m1 string, m2 string) (res bool){
    res = m1 == m2
    return res
}

func IsApproximate(m1 BioString, m2 string, cost int, maxCost int, in int, del int, maxIndel int) ([][4]int){
    indel := in + del
    logger.Debugf("Start IsApproximate cost: %d, in %d, del %d", cost, in, del)
    m1Len := len(m1.GetValue())
    m2Len := len(m2)
    logger.Debugf("Part1:%d %s", m1Len, m1.GetValue())
    logger.Debugf("Part2:%d %s", m2Len, m2)


    results := make([][4]int, 0)
    if m1Len == 0 && m2Len == 0 {
        logger.Debugf("End of comparison, Match! %d;%d;%d", cost,  in, del)
        results = append(results, [4]int{m1Len, cost, in, del})
        return results
        //return true, cost, indel
    }
    if m1Len == 0 && m2Len != 0 {
        if indel >= maxIndel {
            logger.Debugf("End of comparison, Match! %d;%d;%d", cost,  in, del)
            results = append(results, [4]int{m1Len, cost, in, del})
            return results
        }
        allowedIndels := maxIndel - indel
        if maxIndel - indel >= m2Len {
            allowedIndels = m2Len
        }

        for i:=0;i<allowedIndels;i++ {
            logger.Debugf("End of comparison, Match! %d;%d;%d", cost,  in + i, del)
            results = append(results, [4]int{m1Len, cost, in + i, del})
        }
        return results


    }
    if m1Len != 0 && m2Len == 0 {
        if indel >= maxIndel {
            logger.Debugf("End of comparison")
            return results
        }
        if maxIndel - indel < m1Len {
            logger.Debugf("End of comparison")
            return results
            //return true, cost, maxIndel
        }else {
            logger.Debugf("End of comparison, Match! %d;%d;%d", cost, in, del + m1Len)
            results = append(results, [4]int{m1Len, cost, in, del + m1Len})
            return results
            //return true, cost, indel + m1Len
        }
    }

    logger.Debugf("Compare %s vs %s", m1.GetValue()[0], m2[0])
    m1Content := m1.GetValue()
    m1.SetValue(m1.GetValue()[0:1])
    //b1 := DnaString{}
    //b1.Value = m1[0:1]
    //b2 := DnaString{}
    //b2.Value = m2[0:1]
    if ! m1.IsExact(m2[0:1]) {
    //if ! IsBioExact(m1, m2[0:1]) {
        logger.Debugf("Cost: %d <? %d", cost, maxCost)
        if cost < maxCost {
            logger.Debugf("Try with cost")
            m1.SetValue(m1Content[1:m1Len])
            tmpRes := IsApproximate(m1, m2[1:m2Len], cost + 1, maxCost, in, del, maxIndel)
            results = append(results, tmpRes...)
        }
    } else {
        logger.Debugf("Equal, continue...")
        m1.SetValue(m1Content[1:m1Len])
        tmpRes := IsApproximate(m1, m2[1:m2Len], cost, maxCost, in, del, maxIndel)
        results = append(results, tmpRes...)
    }
    if indel < maxIndel {
        logger.Debugf("Try with indel")
        m1.SetValue(m1Content[0:m1Len])
        tmpRes := IsApproximate(m1, m2[1:m2Len], cost, maxCost, in + 1, del, maxIndel)
        results = append(results, tmpRes...)
        m1.SetValue(m1Content[1:m1Len])
        tmpRes = IsApproximate(m1, m2[0:m2Len], cost, maxCost, in, del + 1, maxIndel)
        results = append(results, tmpRes...)
    }
    logger.Debugf("End of comparison")
    return results

}


// Find an exact pattern in sequence
func (s SearchUtils) FindExact(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) {
    // seq := Sequence{grammar.Sequence, 0, ""}
    curVariable := grammar.Models[model].Vars[modelVariable]
    if (curVariable.Value == "" &&
        curVariable.String_constraints.Content != "") {
        contentConstraint := curVariable.String_constraints.Content
        logger.Debugf("FindExact, get var content %s", contentConstraint)
        curVariable.Value = s.SequenceHandler.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
        logger.Debugf("? %s",curVariable.Value)
        if curVariable.Value == "" {
            close(mch)
            return
        }
    }

    logger.Debugf("Search %s at min pos %d, spacer: %t", curVariable.Value, match.MinPosition, spacer)

    findResults := make([][2]int, 0)
    seqLen := s.SequenceHandler.Sequence.Size
    //sequence := seq.GetSequence()
    //seqLen := len(sequence)
    patternLen := len(curVariable.Value)
    minStart := match.MinPosition
    maxStart := match.MinPosition + 1
    if match.Spacer {
        maxStart = seqLen - patternLen + 1
    }
    // log.Printf("seach between %d and %d", minStart, maxStart)
    for i:=minStart; i < maxStart; i++ {
        seqPart := s.SequenceHandler.GetContent(i, i + patternLen)

        bioString := NewDnaString(curVariable.Value)
        if match.Reverse {
            curVariable.Value = bioString.Reverse()
        }
        if curVariable.HasMorphism() {
            bioString.SetMorphisms(curVariable.GetMorphism(grammar.Morphisms).Morph)
        }
        // seqPart := sequence[i:i+patternLen]
        b1 := NewDnaString(curVariable.Value)
        if IsBioExact(&b1, seqPart) {
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
                logger.Debugf("skip match at wrong position: %d" , startResult)
                ban += 1
                continue
            }
        } else {
            if startResult < match.MinPosition {
                logger.Debugf("skip match at wrong position: %d" , startResult)
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
            logger.Debugf("got match: %d, %d", newMatch.Start, newMatch.End)
        }
    }
    logger.Debugf("got matches: %d", (len(findResults) - ban))
    close(mch)
    //return matches
}


func (s SearchUtils) PostControl(match logol.Match, grammar logol.Grammar, contextVars map[string]logol.Match) (newMatch logol.Match, err bool){
    // TODO
    // check model global constraints
    // Check for negative_constraints
    newMatch = match
    logger.Debugf("PostControl checks")

    curVariable := grammar.Models[match.Model].Vars[match.Id]
    if curVariable.HasStartConstraint(){
        logger.Debugf("Control start")
        minS, maxS := curVariable.GetStartConstraint()
        min, _ := utils.GetRangeValue(minS, contextVars)
        max, _ := utils.GetRangeValue(maxS, contextVars)
        if (min != -1 && match.Start < min) || (max != -1 && match.Start > max) {
            return newMatch, true
        }
    }
    if curVariable.HasEndConstraint(){
        logger.Debugf("Control end")
        minS, maxS := curVariable.GetEndConstraint()
        min, _ := utils.GetRangeValue(minS, contextVars)
        max, _ := utils.GetRangeValue(maxS, contextVars)
        if (min != -1 && match.End < min) || (max != -1 && match.End > max) {
            return newMatch, true
        }
    }

    if curVariable.HasCostConstraint(){
        logger.Debugf("Control cost")
        minS, maxS := curVariable.GetCostConstraint()
        min, _ := utils.GetRangeValue(minS, contextVars)
        max, _ := utils.GetRangeValue(maxS, contextVars)
        if (min != -1 && match.Sub < min) || (max != -1 && match.Sub > max) {
            return newMatch, true
        }
    }

    if curVariable.HasDistanceConstraint(){
        logger.Debugf("Control distance")
        minS, maxS := curVariable.GetDistanceConstraint()
        min, _ := utils.GetRangeValue(minS, contextVars)
        max, _ := utils.GetRangeValue(maxS, contextVars)
        if (min != -1 && match.Indel < min) || (max != -1 && match.Indel > max) {
            return newMatch, true
        }
    }

    seqPart := s.SequenceHandler.GetContent(match.Start, match.End)

    if curVariable.HasPercentConstraint(){
        logger.Debugf("Control percent of alphabet")
        alphabet, percent, _ := curVariable.GetPercentConstraint()
        doMatch, _ := utils.CheckAlphabetPercent(seqPart, alphabet, percent)
        if ! doMatch {
            return newMatch, true
        }
    }



    logger.Debugf("Check negative constraints")
    negConstraints := curVariable.Negative_constraints
    if len(negConstraints) > 0 {
        for _, negConstraint := range negConstraints {
            if negConstraint.Value == "" {
                contentConstraint := negConstraint.String_constraints.Content
                negConstraint.Value = s.SequenceHandler.GetContent(contextVars[contentConstraint].Start, contextVars[contentConstraint].End)
            }
            b1 := DnaString{}
            b1.Value = negConstraint.Value
            logger.Debugf("Has negative constraint, check %s against %s", seqPart, b1.Value)
            if IsBioExact(&b1, seqPart) {
                return newMatch, true
            }
        }
    }
    return newMatch, false
}
