package logol

import (
        "gopkg.in/yaml.v2"
)

type Event struct {
    Step int
}

type Result struct {
    SequenceFile string
    RunIndex int
    Uid string
    MsgTo string
    Model string
    ModelVariable string
    From []string
    Matches []Match
    PrevMatches [][]Match // store previous main model matches when using serial models
    ContextVars []map[string]Match
    Spacer bool
    Context [][]Match
    Step int
    Position int
    Iteration int
    Param []Match
}
func NewResult() Result {
    result := Result{}
    result.From = make([]string, 0)
    result.Matches = make([]Match, 0)
    result.PrevMatches = make([][]Match, 0)
    result.ContextVars = make([]map[string]Match, 0)
    result.Context = make([][]Match, 0)
    result.Param = make([]Match, 0)
    return result
}
func (m Result) Dumps() (out []byte, err error) {
    return yaml.Marshal(&m)
}
