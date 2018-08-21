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
type StringConstraint struct {
    Content string
    SaveAs string
    Size SizeConstraint
    Morphism string // name of morphism
    Reverse bool
}
type StructConstraint struct {
    Cost string // range [3,4]
    Distance string // range [10,20]
    Percent string // %ac:80 80% of ac
}
// Define additional constraint on match
type NegConstraint struct {
    Value string
    String_constraints StringConstraint
    Struct_constraints StructConstraint
}
type Variable struct {
    Value string
    Next []string
    Model VariableModel
    String_constraints StringConstraint
    Negative_constraints[] NegConstraint
    Overlap bool
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
