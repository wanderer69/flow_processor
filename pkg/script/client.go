package script

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

// простой скрипт

const (
	LexemaDollar            string = "Dollar"
	LexemaCurlyBracketOpen  string = "CurlyBracketOpen"
	LexemaCurlyBracketClose string = "CurlyBracketClose"
	LexemaParenthesisOpen   string = "ParenthesisOpen"
	LexemaParenthesisClose  string = "ParenthesisClose"
	LexemaNumber            string = "Number"
	LexemaOperator          string = "Operator"
	LexemaString            string = "String"
	LexemaIdentificator     string = "Identificator"
)

type Lexema struct {
	Lexema   string
	Value    string
	Operator string
}

const (
	ExecuteResult string = "result"
)

func ParserLexema(s string) ([]*Lexema, error) {
	lexemas := []*Lexema{}
	var currentLexema *Lexema
	isOperator := func() bool {
		if currentLexema != nil {
			if currentLexema.Lexema == LexemaIdentificator {
				switch strings.ToLower(currentLexema.Value) {
				case "if":
					currentLexema.Operator = strings.ToLower(currentLexema.Value)
					currentLexema.Lexema = LexemaOperator
				}
			}
			lexemas = append(lexemas, currentLexema)
			currentLexema = nil
		}
		return false
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		//fmt.Printf("%d\t%c\n", i, r)
		i += size
		switch string(r) {
		case " ":
			isOperator()
		case "$":
			isOperator()
			lexemas = append(lexemas, &Lexema{Lexema: LexemaDollar})
		case "{":
			isOperator()
			lexemas = append(lexemas, &Lexema{Lexema: LexemaCurlyBracketOpen})
		case "}":
			isOperator()
			lexemas = append(lexemas, &Lexema{Lexema: LexemaCurlyBracketClose})
		case "(":
			isOperator()
			lexemas = append(lexemas, &Lexema{Lexema: LexemaParenthesisOpen})
		case ")":
			isOperator()
			lexemas = append(lexemas, &Lexema{Lexema: LexemaCurlyBracketClose})
		default:
			if unicode.IsDigit(r) {
				if currentLexema == nil {
					currentLexema = &Lexema{
						Lexema: LexemaNumber,
					}
				}
				if currentLexema.Lexema != LexemaNumber {
					if currentLexema.Lexema != LexemaIdentificator {
						lexemas = append(lexemas, currentLexema)
						currentLexema = &Lexema{
							Lexema: LexemaNumber,
						}
					} else {
						currentLexema.Lexema = LexemaIdentificator
					}
				}
				currentLexema.Value += string(r)
				continue
			}
			if unicode.IsLetter(r) {
				if currentLexema == nil {
					currentLexema = &Lexema{
						Lexema: LexemaIdentificator,
					}
				}
				if currentLexema.Lexema != LexemaIdentificator {
					lexemas = append(lexemas, currentLexema)
					currentLexema = &Lexema{
						Lexema: LexemaIdentificator,
					}
				}
				currentLexema.Value += string(r)
				continue
			}
			if unicode.IsSymbol(r) {
			}
		}
	}
	return lexemas, nil
}

const (
	PatternTypeConst    string = "const"
	PatternTypeBlock    string = "block"
	PatternTypeVariable string = "variable"
	PatternTypeLink     string = "link"
)

type PatternItem struct {
	Type     string
	Lexemas  []*Lexema
	Link     string
	Variable *entity.Variable
}

type Pattern struct {
	Name         string
	PatternItems []*PatternItem
}

type UsedPatternItem struct {
	Pattern   *Pattern
	Variables []*entity.Variable
}

type UsedPattern struct {
	UsedPatterns []*UsedPatternItem
}

var patterns []*Pattern = []*Pattern{
	{
		Name: "Begin",
		PatternItems: []*PatternItem{
			{
				Type: PatternTypeConst,
				Lexemas: []*Lexema{
					{Lexema: LexemaDollar, Value: "", Operator: ""},
				},
			},
		},
	},
	{
		Name: "Block",
		PatternItems: []*PatternItem{
			{
				Type: PatternTypeBlock,
				Lexemas: []*Lexema{
					{Lexema: LexemaCurlyBracketOpen, Value: "", Operator: ""},
				},
			},
			/*
				{
					Type: PatternTypeLink,
					Link: "Variable",
				},
			*/
			{
				Type: PatternTypeConst,
				Lexemas: []*Lexema{
					{Lexema: LexemaIdentificator, Value: "", Operator: ""},
				},
				Variable: &entity.Variable{
					Name: "VarName",
				},
			},
			{
				Type: PatternTypeBlock,
				Lexemas: []*Lexema{
					{Lexema: LexemaCurlyBracketClose, Value: "", Operator: ""},
				},
			},
		},
	},
	{
		Name: "Variable",
		PatternItems: []*PatternItem{
			{
				Type: PatternTypeConst,
				Lexemas: []*Lexema{
					{Lexema: LexemaIdentificator, Value: "", Operator: ""},
				},
			},
		},
	},
}

func TranslateLexemaList(ll []*Lexema, context *entity.Context) ([]*entity.Variable, error) {
	usedPattern := &UsedPattern{}
	vars := []*entity.Variable{}
	//  ищем последовательно подходящие паттерны
	var currentPattern *Pattern
	state := 0
	i := 0
	currentPatternCnt := 0
	currentPatternItemsCnt := 0
	currentLexemaInPatternItem := 0
	oldPos := 0
	isStopped := false
	for {
		//fmt.Printf("state %v\r\n", state)
		switch state {
		case 0:
			if currentPattern == nil {
				currentPatternCnt = 0
			}
			currentPattern = patterns[currentPatternCnt]
			currentPatternItemsCnt = 0
			currentLexemaInPatternItem = 0
			oldPos = i
			state = 1
		case 1:
			if currentPattern.PatternItems[currentPatternItemsCnt].Type == PatternTypeLink {

			}
			if ll[i].Lexema == currentPattern.PatternItems[currentPatternItemsCnt].Lexemas[currentLexemaInPatternItem].Lexema {
				switch currentPattern.PatternItems[currentPatternItemsCnt].Lexemas[currentLexemaInPatternItem].Lexema {
				case LexemaIdentificator:
					state = 4
					continue
				default:
					if ll[i].Value == currentPattern.PatternItems[currentPatternItemsCnt].Lexemas[currentLexemaInPatternItem].Value {
						state = 4
						continue
					} else {
						state = 2
					}
				}
			} else {
				state = 2
			}
		case 2:
			i = oldPos
			currentPatternCnt++
			if currentPatternCnt < len(patterns) {
				state = 0
				vars = []*entity.Variable{}
				continue
			}
			isStopped = true
		case 3:
			//fmt.Printf("name %v\r\n", currentPattern.Name)
			usedPattern.UsedPatterns = append(usedPattern.UsedPatterns, &UsedPatternItem{
				Pattern:   currentPattern,
				Variables: vars,
			})
			vars = []*entity.Variable{}
			i++
			if i < len(ll) {
				state = 0
				currentPattern = nil
				vars = []*entity.Variable{}
				continue
			}
			isStopped = true
		case 4:
			currentLexemaInPatternItem++
			if currentLexemaInPatternItem < len(currentPattern.PatternItems[currentPatternItemsCnt].Lexemas) {
				i++
				if i < len(ll) {
					state = 2
				}
			} else {
				if currentPattern.PatternItems[currentPatternItemsCnt].Variable != nil {
					vars = append(vars, &entity.Variable{
						Name:  currentPattern.PatternItems[currentPatternItemsCnt].Variable.Name,
						Value: ll[i].Value,
					})
				}
				currentPatternItemsCnt++
				if currentPatternItemsCnt < len(currentPattern.PatternItems) {
					currentLexemaInPatternItem = 0
					state = 1
					i++
					if i >= len(ll) {
						state = 2
					}
					continue
				}
				state = 3
			}
		}
		if isStopped {
			break
		}
	}

	variables := []*entity.Variable{}
	for i := range usedPattern.UsedPatterns {
		for j := range usedPattern.UsedPatterns[i].Variables {
			//fmt.Printf("%v\r\n", usedPattern.UsedPatterns[i].Variables[j])
			isResult := false
			v, ok := context.VariablesByName[usedPattern.UsedPatterns[i].Variables[j].Value]
			if ok {
				switch v.Type {
				case "boolean":
					vb, err := strconv.ParseBool(v.Value)
					if err != nil {
						fmt.Printf("failed convertion bool %v: %v", v.Value, err)
						vb = false
					}
					isResult = vb
				}
			}
			variables = append(variables, &entity.Variable{
				Name:  ExecuteResult,
				Type:  "boolean",
				Value: fmt.Sprintf("%v", isResult),
			})
		}
	}
	return variables, nil
}