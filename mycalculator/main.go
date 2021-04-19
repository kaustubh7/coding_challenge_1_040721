package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

var globalResult float64 = 0
var numStack []string
var prevResults []PrevResult

type Result struct {
	PreviousResult []PrevResult
	FinalResult    float64
	Message        string
}

type PrevResult struct {
	Input string
	Total float64
}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	if len(request.Body) > 0 && request.Body != "clear" {
		return writeToResponse(cleanUpAndCreateResponseJson(string(request.Body), eval(string(request.Body))))
	} else if request.Body == "clear" {
		return writeToResponse(cleanUpAndCreateResponseJson(string(request.Body), 0))
	} else {
		return writeToResponse(Result{})
	}
}

func eval(inputExpression string) float64 {
	var operator = "+"
	for i := 0; i < len(inputExpression); i++ {
		var char string = string(inputExpression[i])

		if isNumber(char) {
			for i+1 < len(inputExpression) && isNumber(string(inputExpression[i+1])) {
				currentNumber, _ := strconv.ParseFloat(char, 64)
				nextNumber, _ := strconv.ParseFloat(string(inputExpression[i+1]), 64)
				char = fmt.Sprintf("%g", currentNumber*10+nextNumber)
				i = i + 1
			}
			result := writeToNumStack(char, operator)
			if result == -1 {
				numStack = numStack[:0]
				break
			}
		} else if char == "(" {
			numStack = append(numStack, operator)
			numStack = append(numStack, "(")
			operator = "+"
		} else if char == ")" {
			var bracketResult float64 = 0
			for i := len(numStack); i >= 0; i-- {
				if numStack[i-1] == "(" {
					operator = numStack[i-2]
					numStack = numStack[:i]
					numStack = numStack[:i-1]
					numStack = numStack[:i-2]
					break
				}
				t, _ := strconv.ParseFloat(numStack[i-1], 64)
				bracketResult += t
			}
			writeToNumStack(fmt.Sprintf("%g", bracketResult), operator)
		} else {
			operator = char
		}
	}

	var result float64 = 0
	for num := range numStack {
		t, _ := strconv.ParseFloat(numStack[num], 64)
		result += t
	}
	globalResult = result + globalResult
	return result
}

func cleanUpAndCreateResponseJson(body string, total float64) Result {
	numStack = numStack[:0]
	if len(body) > 0 && body != "clear" {
		prevResult := PrevResult{string(body), total}
		prevResults = append(prevResults, prevResult)
		return Result{prevResults, globalResult, "Please enter next expression to calculate"}
	} else {
		prevResults = prevResults[:0]
		globalResult = 0
		return Result{prevResults, globalResult, "history is cleared!"}
	}
}

func writeToResponse(result Result) (events.APIGatewayProxyResponse, error) {
	if result.Message != "" {
		jsonRes, err := json.Marshal(result)
		if err != nil {
			return events.APIGatewayProxyResponse{Body: "Server error occured", StatusCode: 500}, nil
		}
		return events.APIGatewayProxyResponse{Body: string(jsonRes), StatusCode: 200}, nil
	} else {
		return events.APIGatewayProxyResponse{Body: "Please check input!", StatusCode: 200}, nil
	}
}

func writeToNumStack(operand string, operator string) int {
	var num float64 = 0
	num, err := strconv.ParseFloat(operand, 64)
	if err == nil {
		if operator == "-" {
			num = -num
			numStack = append(numStack, fmt.Sprintf("%g", num))
		} else if operator == "*" {
			topNum, _ := strconv.ParseFloat(numStack[len(numStack)-1], 64)
			if err == nil {
				numStack = numStack[:len(numStack)-1]
				numStack = append(numStack, fmt.Sprintf("%g", topNum*num))
			}
		} else if operator == "/" {
			topNum, _ := strconv.ParseFloat(numStack[len(numStack)-1], 64)
			if (topNum == 1 && num == 0) || (topNum == 0 && num == 0) {
				return -1
			}
			if err == nil {
				numStack = numStack[:len(numStack)-1]
				numStack = append(numStack, fmt.Sprintf("%g", topNum/num))
			}
		} else {
			numStack = append(numStack, fmt.Sprintf("%g", num))
		}
	}
	return 0
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func main() {
	lambda.Start(Handler)
}
