package gptSummary

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func SummaryByGemini(content string) (string, error) {
	// set up client
	geminiApiKey := os.Getenv("gemini_api_key")
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		return "", fmt.Errorf("genai.NewClient error: %v", err)
	}
	defer client.Close()

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

	// create messages and limit to be smaller than 8192 tokens
	model := client.GenerativeModel("gemini-1.5-pro")
	respToken, err := model.CountTokens(ctx, genai.Text(contentWithPrompt))
	if err != nil {
		return "", fmt.Errorf("EncodingForModel: %v", err)
	}

	for respToken.TotalTokens > 8192 {
		contentWithPrompt = contentWithPrompt[:len(contentWithPrompt)/10*9]
		respToken, err = model.CountTokens(ctx, genai.Text(contentWithPrompt))
		if err != nil {
			return "", fmt.Errorf("EncodingForModel: %v", err)
		}
	}

	// generate summary
	resp, err := model.GenerateContent(ctx, genai.Text(contentWithPrompt))
	if err != nil {
		return "", fmt.Errorf("client.CreateChatCompletion error: %v", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("client.CreateChatCompletion: no candidates returned")
	}

	resPart := resp.Candidates[0].Content.Parts[0]
	res := fmt.Sprintf("%v", resPart)

	return res, nil
}
