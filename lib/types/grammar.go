package logol

import (
        "gopkg.in/yaml.v2"
)


type Morphism struct {
    Morph map[string][]string
}
type ParamDef struct {
    Inputs []string
    Outputs []string
}
type VariableModel struct {
    Name string
    Param []string
    RepeatMin int
    RepeatMax int
}
type SizeConstraint struct {
    Min string
    Max string
}
type SConstraint struct {
    Content string
    SaveAs string
    Size SizeConstraint
}
// Define additional constraint on match
type NegConstraint struct {
    Value string
    String_constraints SConstraint
}
type Variable struct {
    Value string
    Next []string
    Model VariableModel
    String_constraints SConstraint
    Negative_constraints[] NegConstraint
}
type Model struct {
    Comment string
    Param []string
    Start[] string
    Vars map[string]Variable
}

type MainModel struct {
    Model string
    Param []string
}

type Grammar struct {
    Morphisms []Morphism
    Models map[string]Model
    Run []MainModel
    Sequence string
}

func LoadGrammar(grammar []byte) (error, Grammar) {
    g := Grammar{}
    err := yaml.Unmarshal(grammar, &g)
    return err, g
}

func (g Grammar) DumpGrammar() (out []byte, err error) {
    return yaml.Marshal(&g)
}
