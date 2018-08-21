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
    if expr[0:2] == "@@" {
        return PROP_END, expr[2:len(expr)-1], false
    }
    if expr[0:1] == "@" {
        return PROP_START, expr[1:len(expr)-1], false
    }
    if expr[0:2] == "$$" {
        return PROP_INDEL, expr[2:len(expr)-1], false
    }
    if expr[0:1] == "$" {
        return PROP_SUBST, expr[1:len(expr)-1], false
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

// Calculate expression
//
// Examples: 12 , 1 + @R1 , @R1 * @R2, etc.
func GetRangeValue(expr string, contextVars map[string]logol.Match) (val int, err bool){
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
