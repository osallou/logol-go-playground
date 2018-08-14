package logol

import (
        "gopkg.in/yaml.v2"
)

type Match struct {
    Id string
    Model string
    Uid string
    MinPosition int
    Start int
    End int
    Sub uint
    Indel uint
    Info string
    Children []Match
    MinRepeat int
    MaxRepeat int
    Duration int64  // Time.UnixNano
}
func NewMatch() Match {
    match := Match{}
    match.Start = -1
    match.End = -1
    return match
}
func (m Match) Dumps() (out []byte, err error) {
    return yaml.Marshal(&m)
}

func (m Match) Clone() (clone Match){
    clone = NewMatch()
    clone.Id = m.Id
    clone.Model = m.Model
    clone.Uid = m.Uid
    clone.MinPosition = m.MinPosition
    clone.Start = m.Start
    clone.End = m.End
    clone.Sub = m.Sub
    clone.Indel = m.Indel
    clone.Info = m.Info
    for _, child := range m.Children {
        clone.Children = append(clone.Children, child.Clone())
    }
    clone.MinRepeat = m.MinRepeat
    clone.MaxRepeat = m.MaxRepeat
    clone.Duration = m.Duration
    return clone
}
