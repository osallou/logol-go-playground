package logol

import (
    //"log"
    //"io/ioutil"
    "os"
    "regexp"
    "strconv"
    "strings"
    "github.com/Knetic/govaluate"
    logol "github.com/osallou/logol-go-playground/lib/types"
    logs "github.com/osallou/logol-go-playground/lib/log"
)

var logger = logs.GetLogger("logol.sequence.utils")

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
const PROP_SIZE = 5



// Find all vars name from an expression
//
// Example: @R1 + @@R2 + 3 will return R1,R2
func GetVarNamesFromExpression(expr string) []string {
    result := make([]string, 0)
    varRefs := GetVarReferencesFromExpression(expr)
    for _, ref := range varRefs {
        _, varName, err := getProperty(ref)
        if ! err {
            result = append(result, varName)
        }
    }
    return result
}

// Find all vars reference from an expression
//
// Example: @R1 + @@R2 + 3 will return @R1,@@R2
func GetVarReferencesFromExpression(expr string) []string {
    re := regexp.MustCompile("[$@#]+\\w+")
    res := re.FindAllString(expr, -1)
    return res
}

func Evaluate(expr string, contextVars map[string]logol.Match) bool {
    logger.Debugf("Evaluate expression: %s", expr)

    re := regexp.MustCompile("[$@#]+\\w+")
    res := re.FindAllString(expr, -1)

    parameters := make(map[string]interface{}, 8)
    varIndex := 0
    for _, val := range res {
        t := strconv.Itoa(varIndex)
        varName := "VAR" + t
        r := strings.NewReplacer(val, varName)
        expr = r.Replace(expr)
        varIndex+=1
        cValue, cerr := getValueFromExpression(val, contextVars)
        if cerr {
            logger.Debugf("Failed to get value from expression %s", val)
            return false
        }
        parameters[varName] = cValue
    }
    logger.Debugf("New expr: %s with params %v", expr, parameters)

    expression, err := govaluate.NewEvaluableExpression(expr);
    if err != nil {
        logger.Errorf("Failed to evaluate expression %s", expr)
        return false
    }
	result, _ := expression.Evaluate(parameters);
    if result == true {
        return true
    }else {
        return false
    }
}

// Search variable in context variables and returns value matching selected property (start, end, cost, ...)
func getPropertyValueFromContext(property int, variable string, contextVars map[string]logol.Match) (res int, err bool) {
    ctxVar, ok := contextVars[variable]
    if ! ok {
        logger.Debugf("Could not find %s in context vars", variable)
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
        case PROP_SIZE:
            return ctxVar.End - ctxVar.Start, false
    }
    return 0, false
}

// Search which property of a variable is expected
//
// Example: @@VAR1 means *end* position of VAR1
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
    if exprLen > 1 && expr[0:1] == "#" {
        return PROP_SIZE, expr[1:len(expr)], false
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

// Search if variable constraints refer to a variable not available in context variables
//
// If some variables not yet defined are found, returns list of variable names and value true, else return false
func HasUndefineContentVar(variable string, contextVars map[string]logol.Match) (hasUndefined bool, undefinedVars []string) {
    if variable == "" {
        return false, undefinedVars
    }
    hasUndefined = false
    undefinedVars = make([]string, 0)
    if contextVars == nil {
        hasUndefined = true
        undefinedVars = append(undefinedVars, variable)
    }
    ctxVar, ok := contextVars[variable]
    if ! ok || ctxVar.Start == -1 || ctxVar.End == -1 {
        hasUndefined = true
        undefinedVars = append(undefinedVars, variable)
    }

    return hasUndefined, undefinedVars
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
    logger.Debugf("Check range and extract values if possible")
    elts := strings.Split(expr, " ")
    if len(elts) == 1 {
        val, err := getValueFromExpression(elts[0], contextVars)
        if err {
            return 0, true
        }
        return val, false

    } else {
        if len(elts) != 3 {
            logger.Debugf("Invalid operation %s", expr)
            return 0, true
        }
        val1, err1 := getValueFromExpression(elts[0], contextVars)
        val2, err2 := getValueFromExpression(elts[2], contextVars)
        if err1 || err2 {
            logger.Debugf("Invalid operation %s", expr)
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

// Check the percentage of alphabet in input sequence against input percent
//
// alphabet is a string of chars to be found in sequence. Function counts them vs number of characters in sequence.
// It then compares the found percentage against input percent expecting a higher value
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
    logger.Debugf("Percent match: %d vs %d", percentMatch, percent)
    if percentMatch >= percent {
        return true, percentMatch
    }
    return false, percentMatch
}


func WriteFlowPlots(uid string, flow map[string]string) {
    f, err := os.Create("logol-" + uid + ".stats.dot")
    if err != nil {
        return
    }
    defer f.Close()
    f.WriteString("digraph g {\n")
    for k, v := range flow {
        elts := strings.Split(k, ".")
        f.WriteString("    " + elts[0] + "_" + elts[1] + " -> " + elts[2] + "_" + elts[3] + " [label=" + v + "]\n")
    }
    f.WriteString("}\n")
}
