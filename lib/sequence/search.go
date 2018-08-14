package logol

import (
    "log"
    "regexp"
    logol "org.irisa.genouest/logol/lib/types"
)


func Find(mch chan logol.Match, grammar logol.Grammar, match logol.Match, model string, modelVariable string, contextVars map[string]logol.Match, spacer bool) (matches []logol.Match) {
    // TODO
    matches = FindExact(mch, grammar, match, model, modelVariable, contextVars, spacer)
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

    r, _ := regexp.Compile("(" + curVariable.Value + ")")
    sequence := seq.GetSequence()
    findResults := r.FindAllStringIndex(sequence, -1)
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
