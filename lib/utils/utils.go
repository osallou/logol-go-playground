package logol

import (
    "log"
    "strconv"
    "strings"
    logol "org.irisa.genouest/logol/lib/types"
)

const OPERATION_PLUS = 0
const OPERATION_MINUS = 1
const OPERATION_MULTIPLY = 2
const OPERATION_DIVIDE = 3
const OPERATION_PERCENT = 4

const PROP_START = 0
const PROP_END = 1
const PROP_SUBST = 2
const PROP_INDEL = 3
const PROP_CONTENT = 4

func getPropertyValueFromContext(property int, variable string, contextVars map[string]logol.Match) (res int, err bool) {
    ctxVar, ok := contextVars[variable]
    if ! ok {
        log.Printf("Could not find %s in context vars", variable)
        return 0, true
    }
    switch property {
        case PROP_START:
            return ctxVar.Start, false
        case PROP_END:
            return ctxVar.End, false
        case PROP_SUBST:
            return ctxVar.Sub, false
        case PROP_INDEL:
            return ctxVar.Indel, false
    }
    return 0, false
}

func getProperty(expr string) (prop int, variable string, err bool) {
    exprLen := len(expr)
    if exprLen > 2 && expr[0:2] == "@@" {
        return PROP_END, expr[2:len(expr)], false
    }
    if exprLen > 1 && expr[0:1] == "@" {
        return PROP_START, expr[1:len(expr)], false
    }
    if exprLen > 2 && expr[0:2] == "$$" {
        return PROP_INDEL, expr[2:len(expr)], false
    }
    if exprLen > 1 && expr[0:1] == "$" {
        return PROP_SUBST, expr[1:len(expr)], false
    }
    if exprLen > 1 && expr[0:1] == "?" {
        return PROP_CONTENT, expr[1:len(expr)], false
    }
    return  0, "", true
}

// Check expression string and get its value
//
// Can be an int or a reference to a variable (e.g. @R1)
func getValueFromExpression(expr string, contextVars map[string]logol.Match) (val int, err bool){
    val, serr := strconv.Atoi(expr)
    if serr == nil {
        return val, false
    }

    prop, variable, err := getProperty(expr)
    if err {
        return 0, err
    }
    return getPropertyValueFromContext(prop, variable, contextVars)
}

// check if an operation contains unknown or not defined variables
func HasUndefinedRangeVars(expr string, contextVars map[string]logol.Match) (hasUndefined bool, undefinedVars []string) {
    if expr == "" {
        return false, undefinedVars
    }
    if expr == "_" {
        return false, undefinedVars
    }
    hasUndefined = false
    undefinedVars = make([]string, 0)
    elts := strings.Split(expr, " ")

    testProp := func (elt string) (bool, string){
        _, variable, err := getProperty(elt)
        if ! err {
            if contextVars == nil {
                return true, variable
            }
            ctxVar, ok := contextVars[variable]
            if ! ok || ctxVar.Start == -1 || ctxVar.End == -1 {
                return true, variable
            }
        }
        return false, ""
    }

    undef, undefVar := testProp(elts[0])
    if undef {
        hasUndefined = true
        undefinedVars = append(undefinedVars, undefVar)
    }

    if len(elts) > 1 {
        undef, undefVar := testProp(elts[2])
        if undef {
            hasUndefined = true
            undefinedVars = append(undefinedVars, undefVar)
        }
    }

    return hasUndefined, undefinedVars
}
// Calculate expression
//
// Examples: 12 , 1 + @R1 , @R1 * @R2, etc.
func GetRangeValue(expr string, contextVars map[string]logol.Match) (val int, err bool){
    if expr == "" {
        return -1, false
    }
    if expr == "_" {
        return -1, false
    }
    log.Printf("Check range and extract values if possible")
    elts := strings.Split(expr, " ")
    if len(elts) == 1 {
        val, err := getValueFromExpression(elts[0], contextVars)
        if err {
            return 0, true
        }
        return val, false

    } else {
        if len(elts) != 3 {
            log.Printf("Invalid operation %s", expr)
            return 0, true
        }
        val1, err1 := getValueFromExpression(elts[0], contextVars)
        val2, err2 := getValueFromExpression(elts[2], contextVars)
        if err1 || err2 {
            log.Printf("Invalid operation %s", expr)
            return 0, true
        }
        switch elts[1] {
        case "+": return val1 + val2, false
        case "-": return val1 - val2, false
        case "*": return val1 * val2, false
        case "/": return val1 / val2, false
        }
    }
    return 0, false
}

func CheckAlphabetPercent(seqPart string, alphabet string, percent int) (bool, int) {
    nbMatch := 0
    seqPartLen := len(seqPart)
    alphalen := len(alphabet)
    for i:=0;i<seqPartLen;i++ {
        for j:=0;j<alphalen;j++ {
            if seqPart[i] == alphabet[j]{
                nbMatch += 1
                break
            }
        }
    }
    percentMatch := nbMatch * 100 / seqPartLen
    log.Printf("Percent match: %d vs %d", percentMatch, percent)
    if percentMatch >= percent {
        return true, percentMatch
    }
    return false, percentMatch
}
