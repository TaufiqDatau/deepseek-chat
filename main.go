package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type RequestBody struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int16     `json:"max_tokens"`
	Stream    bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCost    float64 `json:"estimated_cost"`
}

type ResponseBody struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

func callDeepInfra(apiKey, userMessage string) (string, error) {
	url := "https://api.deepinfra.com/v1/openai/chat/completions"

	requestBody := RequestBody{
		Model:     "deepseek-ai/DeepSeek-R1",
		Stream:    true,
		MaxTokens: 10000,
		Messages: []Message{
			{Role: "user", Content: userMessage},
		},
	}

	log.Println("request body :", requestBody)

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	result := handleStreamInput(resp)

	return result, nil
}

type streamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string      `json:"role"`
			Content   string      `json:"content"`
			ToolCalls interface{} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason interface{} `json:"finish_reason"`
	} `json:"choices"`
	Usage interface{} `json:"usage"`
}

func handleStreamInput(resp *http.Response) string {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println(resp)
		return ""
	}

	scanner := bufio.NewScanner(resp.Body)

	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(line[len("data:"):])
		if data == "" {
			continue
		}

		if strings.Contains(line, "[DONE]") {
			log.Println("\nReceived [DONE] marker. Stream finished.")
			break
		}

		var sr streamResponse

		if err := json.Unmarshal([]byte(data), &sr); err != nil {
			fmt.Println()
			fmt.Println(line)
			fmt.Println("Error unmarshalling JSON:", err)
			continue
		}

		if sr.Choices[0].FinishReason != nil {
			fmt.Println()
			log.Println("The response is finish")
			continue
		}

		// Append token text to the complete response and print it
		content := sr.Choices[0].Delta.Content
		fullResponse.WriteString(content)
		fmt.Print(content)
	}

	return fullResponse.String()
}

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	apiKey := os.Getenv("DEEPINFRA_API_KEY") // Store API key in environment variable
	if apiKey == "" {
		fmt.Println("API key is missing. Set DEEPINFRA_API_KEY environment variable.")
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter your question (type 'quit' or 'exit' to stop):")

	// Loop to continuously read input from the console
	for scanner.Scan() {
		input := scanner.Text()
		if input == "quit" || input == "exit" {
			fmt.Println("Exiting...")
			break
		}

		_, err := callDeepInfra(apiKey, input)
		if err != nil {
			fmt.Println("Error:", err)
		}
		fmt.Println("\nEnter your next question (type 'quit' or 'exit' to stop):")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}

}
