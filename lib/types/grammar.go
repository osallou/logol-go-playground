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
type SConstraint struct {
    Content string
    SaveAs string
}
type Variable struct {
    Value string
    Next []string
    Model VariableModel
    String_constraints SConstraint
}
type Model struct {
    Comment string
    Param []string
    Start string
    Vars map[string]Variable
}
type Grammar struct {
    Morphisms []Morphism
    Models map[string]Model
    Run []string
}

func LoadGrammar(grammar []byte) (error, Grammar) {
    g := Grammar{}
    err := yaml.Unmarshal(grammar, &g)
    return err, g
}

func (g Grammar) DumpGrammar() (out []byte, err error) {
    return yaml.Marshal(&g)
}
