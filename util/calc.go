package util

import (
	"fmt"

	"github.com/Knetic/govaluate"
)

// 중위 표기법 수식을 후위 표기법으로 변환하는 함수

// Calcuate evaluates a string arithmetic expression like "100 * (123 / 123123)"
func Calcuate(expression string) (float64, error) {
	expr, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %v", err)
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return 0, fmt.Errorf("evaluation error: %v", err)
	}

	floatResult, ok := result.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}

	return floatResult, nil
}

// func evaluateExpression(expression string) (float64, error) {
// 	expression = strings.ReplaceAll(expression, " ", "")
// 	tokens := tokenize(expression)

// 	operatorStack := []string{}
// 	operandStack := []float64{}

// 	for i := 0; i < len(tokens); i++ {
// 		token := tokens[i]

// 		if isDigit(token) {
// 			operand, err := strconv.ParseFloat(token, 64)
// 			if err != nil {
// 				return 0, fmt.Errorf("Invalid expression")
// 			}
// 			operandStack = append(operandStack, operand)
// 		} else if token == "-" && (i == 0 || isOperator(tokens[i-1]) || tokens[i-1] == "(") {
// 			// 음수 처리
// 			nextToken := tokens[i+1]
// 			if !isDigit(nextToken) {
// 				return 0, fmt.Errorf("Invalid expression")
// 			}
// 			operand, err := strconv.ParseFloat(nextToken, 64)
// 			if err != nil {
// 				return 0, fmt.Errorf("Invalid expression")
// 			}
// 			operandStack = append(operandStack, -operand)
// 			i++ // 다음 토큰 건너뛰기
// 		} else if isOperator(token) {
// 			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] != "(" && hasHigherPrecedence(operatorStack[len(operatorStack)-1], token) {
// 				operator := operatorStack[len(operatorStack)-1]
// 				operatorStack = operatorStack[:len(operatorStack)-1]

// 				if len(operandStack) < 2 {
// 					return 0, fmt.Errorf("Invalid expression")
// 				}
// 				operand2 := operandStack[len(operandStack)-1]
// 				operand1 := operandStack[len(operandStack)-2]
// 				operandStack = operandStack[:len(operandStack)-2]

// 				result, err := performOperation(operator, operand1, operand2)
// 				if err != nil {
// 					return 0, err
// 				}
// 				operandStack = append(operandStack, result)
// 			}

// 			operatorStack = append(operatorStack, token)
// 		} else if token == "(" {
// 			operatorStack = append(operatorStack, token)
// 		} else if token == ")" {
// 			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] != "(" {
// 				operator := operatorStack[len(operatorStack)-1]
// 				operatorStack = operatorStack[:len(operatorStack)-1]

// 				if len(operandStack) < 2 {
// 					return 0, fmt.Errorf("Invalid expression")
// 				}
// 				operand2 := operandStack[len(operandStack)-1]
// 				operand1 := operandStack[len(operandStack)-2]
// 				operandStack = operandStack[:len(operandStack)-2]

// 				result, err := performOperation(operator, operand1, operand2)
// 				if err != nil {
// 					return 0, err
// 				}
// 				operandStack = append(operandStack, result)
// 			}

// 			if len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] == "(" {
// 				operatorStack = operatorStack[:len(operatorStack)-1]
// 			}
// 		}
// 	}

// 	for len(operatorStack) > 0 {
// 		operator := operatorStack[len(operatorStack)-1]
// 		operatorStack = operatorStack[:len(operatorStack)-1]

// 		if len(operandStack) < 2 {
// 			return 0, fmt.Errorf("Invalid expression")
// 		}
// 		operand2 := operandStack[len(operandStack)-1]
// 		operand1 := operandStack[len(operandStack)-2]
// 		operandStack = operandStack[:len(operandStack)-2]

// 		result, err := performOperation(operator, operand1, operand2)
// 		if err != nil {
// 			return 0, err
// 		}
// 		operandStack = append(operandStack, result)
// 	}

// 	if len(operandStack) != 1 {
// 		return 0, fmt.Errorf("Invalid expression")
// 	}
// 	return operandStack[0], nil
// }

// func tokenize(expression string) []string {
// 	tokens := []string{}
// 	currentNumber := ""

// 	for _, char := range expression {
// 		if isDigit(string(char)) || char == '.' {
// 			currentNumber += string(char)
// 		} else if isOperator(string(char)) {
// 			if currentNumber != "" {
// 				tokens = append(tokens, currentNumber)
// 				currentNumber = ""
// 			}
// 			tokens = append(tokens, string(char))
// 		} else if char == '(' || char == ')' {
// 			if currentNumber != "" {
// 				tokens = append(tokens, currentNumber)
// 				currentNumber = ""
// 			}
// 			tokens = append(tokens, string(char))
// 		}
// 	}

// 	if currentNumber != "" {
// 		tokens = append(tokens, currentNumber)
// 	}

// 	return tokens
// }

// func isDigit(token string) bool {
// 	_, err := strconv.ParseFloat(token, 64)
// 	return err == nil
// }

// func isOperator(token string) bool {
// 	return token == "+" || token == "-" || token == "*" || token == "/"
// }

// func hasHigherPrecedence(op1, op2 string) bool {
// 	return (op1 == "*" || op1 == "/") && (op2 == "+" || op2 == "-")
// }

// func performOperation(operator string, operand1, operand2 float64) (float64, error) {
// 	switch operator {
// 	case "+":
// 		return operand1 + operand2, nil
// 	case "-":
// 		return operand1 - operand2, nil
// 	case "*":
// 		return operand1 * operand2, nil
// 	case "/":
// 		if operand2 == 0 {
// 			return 0, fmt.Errorf("Divide by zero error")
// 		}
// 		return operand1 / operand2, nil
// 	default:
// 		return 0, fmt.Errorf("Invalid operator")
// 	}
// }
