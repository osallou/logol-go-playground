// PAckage utils define utilitary functions used in different packages
package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	logs "github.com/osallou/logol-go-playground/lib/log"
	transport "github.com/osallou/logol-go-playground/lib/transport"
	logol "github.com/osallou/logol-go-playground/lib/types"
)

var logger = logs.GetLogger("logol.sequence.utils")

/*
const OPERATION_PLUS = 0
const OPERATION_MINUS = 1
const OPERATION_MULTIPLY = 2
const OPERATION_DIVIDE = 3
const OPERATION_PERCENT = 4
*/

const propStart = 0
const propEnd = 1
const propSubst = 2
const propIndel = 3
const propContent = 4
const propSize = 5

// FilterOut removes matches to ban (related to negative model matches) and duplicate results
func FilterOut(t transport.Transport, filePath string, toBan []string) (filtered int, duplicates int, nbMatch int) {
	for i := 0; i < len(toBan); i++ {
		logger.Debugf("Should ban: %s", toBan[i])
	}

	tmpfile, err := ioutil.TempFile("", t.GetID())
	logger.Debugf("Write filtered file: %s", tmpfile.Name())
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	filtered = 0
	duplicates = 0
	nbMatch = 0
	scanner := bufio.NewScanner(file)
	prevMatch := ""
	for scanner.Scan() {
		match := [][]logol.Match{}
		jsonMatch := scanner.Text()
		if jsonMatch == prevMatch {
			duplicates++
			continue
		}
		prevMatch = jsonMatch
		// fmt.Println(jsonMatch)
		err := json.Unmarshal([]byte(jsonMatch), &match)
		if err != nil {
			log.Printf("Failed to load a match, skipping it: %s", jsonMatch)
			continue
		}
		mustBeRemoved := false
		for i := 0; i < len(match); i++ {
			smatch := match[i]
			for j := 0; j < len(smatch); j++ {
				ssmatch := smatch[j]
				for t := 0; t < len(toBan); t++ {
					mToBan := toBan[t]
					_, found := ssmatch.GetByUid(mToBan)
					if found {
						mustBeRemoved = true
						break
					}
				}
				if mustBeRemoved {
					break
				}
			}
			if mustBeRemoved {
				break
			}
		}
		if !mustBeRemoved {
			fmt.Fprintln(tmpfile, "", jsonMatch)
			nbMatch++
		} else {
			filtered++
		}
	}
	os.Rename(tmpfile.Name(), filePath)
	return filtered, duplicates, nbMatch
}

// GetSize returns size of global match
func GetSize(matches []logol.Match) int {
	start := 0
	end := 0
	if len(matches) > 0 {
		start = matches[0].Start
		end = matches[len(matches)-1].End
	}
	return end - start
}

// GetVarNamesFromExpression find all vars name from an expression
//
// Example: @R1 + @@R2 + 3 will return R1,R2
func GetVarNamesFromExpression(expr string) []string {
	result := make([]string, 0)
	varRefs := GetVarReferencesFromExpression(expr)
	for _, ref := range varRefs {
		_, varName, err := getProperty(ref)
		if !err {
			result = append(result, varName)
		}
	}
	return result
}

// GetVarReferencesFromExpression find all vars reference from an expression
//
// Example: @R1 + @@R2 + 3 will return @R1,@@R2
func GetVarReferencesFromExpression(expr string) []string {
	re := regexp.MustCompile("[$@#]+\\w+")
	res := re.FindAllString(expr, -1)
	return res
}

// Evaluate takes a math expression and applies it based on input contextVars
func Evaluate(expr string, contextVars map[string]logol.Match) bool {
	logger.Debugf("Evaluate expression: %s", expr)

	re := regexp.MustCompile("[$@#]+\\w+")
	res := re.FindAllString(expr, -1)
	// msg, _ := json.Marshal(contextVars)
	// logger.Errorf("CONTEXT: %s", msg)
	parameters := make(map[string]interface{}, 8)
	varIndex := 0
	for _, val := range res {
		t := strconv.Itoa(varIndex)
		varName := "VAR" + t
		r := strings.NewReplacer(val, varName)
		expr = r.Replace(expr)
		varIndex++
		cValue, cerr := getValueFromExpression(val, contextVars)
		if cerr {
			logger.Debugf("Failed to get value from expression %s", val)
			return false
		}
		parameters[varName] = cValue
	}
	logger.Debugf("New expr: %s with params %v", expr, parameters)

	expression, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		logger.Errorf("Failed to evaluate expression %s", expr)
		return false
	}
	result, _ := expression.Evaluate(parameters)
	if result == true {
		return true
	}
	return false
}

// Search variable in context variables and returns value matching selected property (start, end, cost, ...)
func getPropertyValueFromContext(property int, variable string, contextVars map[string]logol.Match) (res int, err bool) {
	ctxVar, ok := contextVars[variable]
	if !ok {
		logger.Debugf("Could not find %s in context vars", variable)
		return 0, true
	}
	switch property {
	case propStart:
		return ctxVar.Start, false
	case propEnd:
		return ctxVar.End, false
	case propSubst:
		return ctxVar.Sub, false
	case propIndel:
		return ctxVar.Indel, false
	case propSize:
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
		return propEnd, expr[2:len(expr)], false
	}
	if exprLen > 1 && expr[0:1] == "@" {
		return propStart, expr[1:len(expr)], false
	}
	if exprLen > 2 && expr[0:2] == "$$" {
		return propIndel, expr[2:len(expr)], false
	}
	if exprLen > 1 && expr[0:1] == "$" {
		return propSubst, expr[1:len(expr)], false
	}
	if exprLen > 1 && expr[0:1] == "?" {
		return propContent, expr[1:len(expr)], false
	}
	if exprLen > 1 && expr[0:1] == "#" {
		return propSize, expr[1:len(expr)], false
	}
	return 0, "", true
}

// Check expression string and get its value
//
// Can be an int or a reference to a variable (e.g. @R1)
func getValueFromExpression(expr string, contextVars map[string]logol.Match) (val int, err bool) {
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

// HasUndefineContentVar search if variable constraints refer to a variable not available in context variables
//
// If some variables not yet defined are found, returns list of variable names and value true, else return false
func HasUndefineContentVar(variable string, contextVars map[string]logol.Match) (hasUndefined bool, undefinedVars []string, knownVars []string) {
	if variable == "" {
		return false, undefinedVars, knownVars
	}
	hasUndefined = false
	undefinedVars = make([]string, 0)
	knownVars = make([]string, 0)
	if contextVars == nil {
		hasUndefined = true
		undefinedVars = append(undefinedVars, variable)
	}
	ctxVar, ok := contextVars[variable]
	if !ok || ctxVar.Start == -1 || ctxVar.End == -1 {
		hasUndefined = true
		undefinedVars = append(undefinedVars, variable)
	} else {
		knownVars = append(knownVars, variable)
	}

	return hasUndefined, undefinedVars, knownVars
}

// HasUndefinedRangeVars check if an operation contains unknown or not defined variables
func HasUndefinedRangeVars(expr string, contextVars map[string]logol.Match) (hasUndefined bool, undefinedVars []string, knownVars []string) {
	hasUndefined = false
	undefinedVars = make([]string, 0)
	knownVars = make([]string, 0)
	if expr == "" {
		return false, undefinedVars, knownVars
	}
	if expr == "_" {
		return false, undefinedVars, knownVars
	}

	elts := strings.Split(expr, " ")

	testProp := func(elt string) (bool, string) {
		_, variable, err := getProperty(elt)
		if !err {
			if contextVars == nil {
				return true, variable
			}
			ctxVar, ok := contextVars[variable]
			if !ok || ctxVar.Start == -1 || ctxVar.End == -1 {
				return true, variable
			}
		}
		return false, variable
	}
	logger.Debugf("testing expr %s", expr)
	undef, undefVar := testProp(elts[0])
	if undef {
		hasUndefined = true
		undefinedVars = append(undefinedVars, undefVar)
	} else {
		if undefVar != "" {
			knownVars = append(knownVars, undefVar)
		}
	}

	if len(elts) > 1 {
		undef, undefVar := testProp(elts[2])
		if undef {
			hasUndefined = true
			undefinedVars = append(undefinedVars, undefVar)
		} else {
			if undefVar != "" {
				knownVars = append(knownVars, undefVar)
			}
		}
	}

	return hasUndefined, undefinedVars, knownVars
}

// GetRangeValue calculate expression
//
// Examples: 12 , 1 + @R1 , @R1 * @R2, etc.
func GetRangeValue(expr string, contextVars map[string]logol.Match) (val int, err bool) {
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

	}
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
	case "+":
		return val1 + val2, false
	case "-":
		return val1 - val2, false
	case "*":
		return val1 * val2, false
	case "/":
		return val1 / val2, false
	}

	return 0, false
}

// CheckAlphabetPercent check the percentage of alphabet in input sequence against input percent
//
// alphabet is a string of chars to be found in sequence. Function counts them vs number of characters in sequence.
// It then compares the found percentage against input percent expecting a higher value
func CheckAlphabetPercent(seqPart string, alphabet string, percent int) (bool, int) {
	nbMatch := 0
	seqPartLen := len(seqPart)
	alphalen := len(alphabet)
	for i := 0; i < seqPartLen; i++ {
		for j := 0; j < alphalen; j++ {
			if seqPart[i] == alphabet[j] {
				nbMatch++
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

// WriteFlowPlots writes stats info (workflow) in DOT format
func WriteFlowPlots(uid string, flow map[string]string, duration map[string]string) {
	doStatsEnv := os.Getenv("LOGOL_STATS")
	if doStatsEnv == "" {
		return
	}
	f, err := os.Create("logol-" + uid + ".stats.dot")
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString("digraph g {\n")
	for k, v := range duration {
		elts := strings.Split(k, ".")
		f.WriteString("    " + elts[0] + "_" + elts[1] + "[label=\"" + k + " - " + v + "\"]\n")
	}
	for k, v := range flow {
		elts := strings.Split(k, ".")
		f.WriteString("    " + elts[0] + "_" + elts[1] + " -> " + elts[2] + "_" + elts[3] + " [label=" + v + "]\n")
	}
	f.WriteString("}\n")
}
