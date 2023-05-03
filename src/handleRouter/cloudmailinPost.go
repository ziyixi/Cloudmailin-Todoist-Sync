package handleRouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	_ "embed"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/ziyixi/Cloudmailin-Todoist-Sync/src/gptSummary"
)

type cloudmailinPost struct {
	From    string // headers.from
	To      string // headers.to
	Date    string // headers.date
	Subject string // headers.subject
	Content string // md(html)
}

func parseCloudmailinJson(s string) *cloudmailinPost {
	converter := md.NewConverter("", true, nil)
	html := gjson.Get(s, "html").String()

	// convert html to markdown
	markdownRaw, err := converter.ConvertString(html)
	if err != nil || len(markdownRaw) == 0 {
		// use plain text instead
		markdownRaw = gjson.Get(s, "plain").String()
	}

	// remove all urls, otherwise there will be too many tokens for next-step processing
	urlPattern := `\(\s*https[^()]*\)`
	m := regexp.MustCompile(urlPattern)
	markdown := m.ReplaceAllString(markdownRaw, "()")

	res := cloudmailinPost{
		From:    gjson.Get(s, "headers.from").String(),
		To:      gjson.Get(s, "headers.to").String(),
		Date:    gjson.Get(s, "headers.date").String(),
		Subject: gjson.Get(s, "headers.subject").String(),
		Content: markdown,
	}

	// Outlook email subject may have a prefix FW:
	heloDomain := gjson.Get(s, "envelope.helo_domain").String()
	if strings.Contains(heloDomain, "outlook") && strings.HasPrefix(res.Subject, "FW: ") {
		res.Subject = res.Subject[4:]
	}

	// Outlook might foward the email in the forwarding format
	if strings.Contains(res.To, "cloudmailin") {
		// parse the correct email address
		re, _ := regexp.Compile(`_+\\r\\nFrom: .*?([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)`)
		matches := re.FindStringSubmatch(s)
		if len(matches) < 2 {
			res.To = res.From
			res.From = "sender unknown"
		} else {
			res.To = res.From
			res.From = matches[1]
		}
	}

	return &res
}

type AddTaskRequestAndResponse struct {
	Content     string   `json:"content"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	Url         string   `json:"url"`
}

//go:embed template/todoistDescription.tmpl
var descriptionTmpl string

func HandleCloudmailinPost(c *gin.Context) {
	// get the post data
	jsonRaw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error in reading json body": err.Error()})
		return
	}
	jsonString := string(jsonRaw)
	emailContent := parseCloudmailinJson(jsonString)
	if len(emailContent.From) == 0 || len(emailContent.To) == 0 || (len(emailContent.Subject) == 0 && len(emailContent.Content) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error in parsing json body": "from/to/subject/content is empty"})
		return
	}

	// summary the email content
	emailContent.Content, err = gptSummary.SummaryByChatGPT(emailContent.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in summarizing email content": err.Error()})
		return
	}

	// create todoist task
	todoistApiKey := os.Getenv("todoist_api_key")

	// prepare task description, load template
	tmpl, err := template.New("todoistDescription").Parse(descriptionTmpl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in loading template": err.Error()})
		return
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, emailContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in loading template": err.Error()})
		return
	}
	taskDescription := buf.String()

	// make post request to todoist
	taskRequestContent := AddTaskRequestAndResponse{
		Content:     fmt.Sprintf("%v [%v]", emailContent.Subject, emailContent.From),
		Labels:      []string{"email"},
		Description: taskDescription,
	}
	client := resty.New()
	unewUUID, err := uuid.NewRandom()
	unewUUIDString := unewUUID.String()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in generating uuid": err.Error()})
		return
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Request-Id", strings.TrimSpace(unewUUIDString)).
		SetHeader("Authorization", "Bearer "+todoistApiKey).
		SetBody(taskRequestContent).
		Post("https://api.todoist.com/rest/v2/tasks")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in creating todoist task": err.Error()})
		return
	}

	// parse url from response
	task := AddTaskRequestAndResponse{}
	err = json.Unmarshal(resp.Body(), &task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in parsing todoist response": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": task.Url})
}
