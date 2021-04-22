package main

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"text/template"

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
	var expression = request.QueryStringParameters["expression"]
	if expression == "clear" {
		return writeToResponse(clearResults())
	} else {
		return writeToResponse(cleanUpAndCreateResponseJson(expression, eval(expression)))
	}
}

func eval(inputExpression string) float64 {
	var operator = "+"
	if len(inputExpression) == 0 {
		return 0
	}
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

func BuildPage(htmlTemplate string, result Result) *bytes.Buffer {
	var bodyBuffer bytes.Buffer
	t := template.New("template")
	var templates = template.Must(t.Parse(htmlTemplate))
	templates.Execute(&bodyBuffer, result)
	return &bodyBuffer
}

func cleanUpAndCreateResponseJson(body string, total float64) Result {
	numStack = numStack[:0]
	isValidExpr := regexp.MustCompile(`[0-9\\+\\-\\*\\/\\(\\)]+$`).MatchString
	if isValidExpr(body) {
		prevResult := PrevResult{string(body), total}
		prevResults = append(prevResults, prevResult)
		return Result{prevResults, globalResult, "Please enter next expression to calculate"}
	} else {
		return Result{prevResults, globalResult, "Enter a valid string to continue"}
	}
}

func clearResults() Result {
	prevResults = prevResults[:0]
	globalResult = 0
	return Result{prevResults, globalResult, "history is cleared!"}

}

func writeToResponse(result Result) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"content-type": "text/html"},
		Body:       BuildPage(HtmlTemplate, result).String(),
		StatusCode: 200}, nil
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

var HtmlTemplate string = `<html>
<head>
	<style type='text/css'>
	@import url('https://fonts.googleapis.com/css?family=Poppins');

	/* BASIC */
	
	html {
	  background-color: #56baed;
	}
	
	body {
	  font-family: 'Poppins', sans-serif;
	  height: 100vh;
	}
	
	a {
	  color: #92badd;
	  display:inline-block;
	  text-decoration: none;
	  font-weight: 400;
	}
	
	h2 {
	  text-align: center;
	  font-size: 16px;
	  font-weight: 600;
	  text-transform: uppercase;
	  display:inline-block;
	  margin: 40px 8px 10px 8px; 
	  color: #cccccc;
	}
	table {
	  text-align: center;
	  font-size: 14px;
	  font-weight: 200;
	  text-transform: uppercase;
	  display:inline-block;
	  margin: 10px 8px 10px 8px; 
	  color: #0d0d0d;
	}
	p {
	  text-align: center;
	  font-size: 14px;
	  font-weight: 200;
	  margin: 40px 8px 10px 8px; 
	  color: #f30b0b;
	}
	
	
	
	/* STRUCTURE */
	
	.wrapper {
	  display: flex;
	  align-items: center;
	  flex-direction: column; 
	  justify-content: center;
	  width: 100%;
	  min-height: 100%;
	  padding: 20px;
	}
	
	#formContent {
	  -webkit-border-radius: 10px 10px 10px 10px;
	  border-radius: 10px 10px 10px 10px;
	  background: #fff;
	  padding: 30px;
	  width: 90%;
	  max-width: 450px;
	  position: relative;
	  padding: 0px;
	  -webkit-box-shadow: 0 30px 60px 0 rgba(0,0,0,0.3);
	  box-shadow: 0 30px 60px 0 rgba(0,0,0,0.3);
	  text-align: center;
	}
	
	#formFooter {
	  background-color: #f6f6f6;
	  border-top: 1px solid #dce8f1;
	  padding: 25px;
	  text-align: center;
	  -webkit-border-radius: 0 0 10px 10px;
	  border-radius: 0 0 10px 10px;
	}
	
	
	
	/* TABS */
	
	h2.inactive {
	  color: #cccccc;
	}
	
	h2.active {
	  color: #0d0d0d;
	  border-bottom: 2px solid #5fbae9;
	}
	
	
	
	/* FORM TYPOGRAPHY*/
	
	input[type=button], input[type=submit], input[type=reset]  {
	  background-color: #56baed;
	  border: none;
	  color: white;
	  padding: 15px 80px;
	  text-align: center;
	  text-decoration: none;
	  display: inline-block;
	  text-transform: uppercase;
	  font-size: 13px;
	  -webkit-box-shadow: 0 10px 30px 0 rgba(95,186,233,0.4);
	  box-shadow: 0 10px 30px 0 rgba(95,186,233,0.4);
	  -webkit-border-radius: 5px 5px 5px 5px;
	  border-radius: 5px 5px 5px 5px;
	  margin: 5px 20px 40px 20px;
	  -webkit-transition: all 0.3s ease-in-out;
	  -moz-transition: all 0.3s ease-in-out;
	  -ms-transition: all 0.3s ease-in-out;
	  -o-transition: all 0.3s ease-in-out;
	  transition: all 0.3s ease-in-out;
	}
	
	input[type=button]:hover, input[type=submit]:hover, input[type=reset]:hover  {
	  background-color: #39ace7;
	}
	
	input[type=button]:active, input[type=submit]:active, input[type=reset]:active  {
	  -moz-transform: scale(0.95);
	  -webkit-transform: scale(0.95);
	  -o-transform: scale(0.95);
	  -ms-transform: scale(0.95);
	  transform: scale(0.95);
	}
	
	input[type=text] {
	  background-color: #f6f6f6;
	  border: none;
	  color: #0d0d0d;
	  padding: 15px 32px;
	  text-align: center;
	  text-decoration: none;
	  display: inline-block;
	  font-size: 16px;
	  margin: 5px;
	  width: 85%;
	  border: 2px solid #f6f6f6;
	  -webkit-transition: all 0.5s ease-in-out;
	  -moz-transition: all 0.5s ease-in-out;
	  -ms-transition: all 0.5s ease-in-out;
	  -o-transition: all 0.5s ease-in-out;
	  transition: all 0.5s ease-in-out;
	  -webkit-border-radius: 5px 5px 5px 5px;
	  border-radius: 5px 5px 5px 5px;
	}
	
	input[type=text]:focus {
	  background-color: #fff;
	  border-bottom: 2px solid #5fbae9;
	}
	
	input[type=text]:placeholder {
	  color: #cccccc;
	  font-size: 12px;
	}            
	
	/* OTHERS */
	
	*:focus {
		outline: none;
	}
	
	* {
	  box-sizing: border-box;
	}
</style>
</head>
<body>
   

	<div class='wrapper fadeIn'>
	<div id='formContent'>
		<!-- Tabs Titles -->
		<h2 class='active'> Calculator </h2>

		<!-- Login Form -->
		<form action='/mycalculator' method='get'>
		<input type='text' id='login' class='fadeIn second' name='expression' placeholder='enter expression here and hit enter'>
		<h4>Previous results</h4>
		{{if .PreviousResult}}
		<table type='text' class='fadeIn second'>{{range $y, $x := .PreviousResult }}
			<tr>
			<td>{{ $x.Input }}</td>
			<td> = </td>
			<td>{{ $x.Total }}</td>
			</tr>{{end}}
		</table>
		<h4>Total</h4>
		<h3 type='text' class='fadeIn second'>{{.FinalResult}}</h3>
		{{end}}
		<p type='text' class='fadeIn second'><b>{{.Message}}</b></p>
		</form>
		<form action='/mycalculator' method='get'>
			<input type='hidden' id='login' class='fadeIn second' name='expression' value='clear'>
			<input type='submit' value='Clear Results'>
		</form>

	</div>
	</div>
</body>
</html>`
