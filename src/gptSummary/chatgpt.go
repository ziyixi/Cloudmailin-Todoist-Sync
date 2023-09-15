package gptSummary

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"

	_ "embed"

	openai "github.com/sashabaranov/go-openai"
)

//go:embed template/prompt.tmpl
var promptTmpl string

func SummaryByChatGPT(content string) (string, error) {
	// set up client
	openaiApiKey := os.Getenv("openai_api_key")
	client := openai.NewClient(openaiApiKey)

	tmpl, err := template.New("prompt").Parse(promptTmpl)
	if err != nil {
		return "", fmt.Errorf("template.ParseFiles error: %v", err)
	}

	// Create a buffer to hold the rendered output
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		EmailContent string
	}{
		EmailContent: content,
	})
	if err != nil {
		return "", fmt.Errorf("tmpl.Execute error: %v", err)
	}
	contentWithPrompt := buf.String()

	// create messages and limit to be smaller than 4000 tokens
	message := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: contentWithPrompt,
		},
	}
	// do binary cut to message.Content to make sure it is smaller than 4000 tokens
	encodedSize, err := NumTokensFromMessages(message, "gpt-3.5-turbo")
	if err != nil {
		return "", fmt.Errorf("EncodingForModel: %v", err)
	}

	for encodedSize > 4000 {
		contentWithPrompt = contentWithPrompt[:len(contentWithPrompt)/10*9]
		message[0].Content = contentWithPrompt
		encodedSize, err = NumTokensFromMessages(message, "gpt-3.5-turbo")
		if err != nil {
			return "", fmt.Errorf("EncodingForModel: %v", err)
		}
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: message,
		},
	)

	if err != nil {
		return "", fmt.Errorf("client.CreateChatCompletion error: %v", err)
	}

	res := resp.Choices[0].Message.Content

	return res, nil
}
