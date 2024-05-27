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
	"time"

	_ "embed"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/ziyixi/Cloudmailin-Todoist-Sync/src/gptSummary"
	"github.com/ziyixi/go-ticktick"
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
	gptmode := os.Getenv("gptmode")
	if gptmode == "chatgpt" {
		emailContent.Content, err = gptSummary.SummaryByChatGPT(emailContent.Content)
	} else if gptmode == "gemini" {
		emailContent.Content, err = gptSummary.SummaryByGemini(emailContent.Content)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error in gptmode": "gptmode is not in .env"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error in summarizing email content": err.Error()})
		return
	}

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

	appmode := os.Getenv("appmode")
	if appmode == "todoist" {
		todoistApiKey := os.Getenv("todoist_api_key")
		// * use todoist
		// make post request to todoist
		taskRequestContent := AddTaskRequestAndResponse{
			Content:     fmt.Sprintf("%v", emailContent.Subject),
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
	} else if appmode == "dida365" {
		// * use dida365 instead
		didaUsername := os.Getenv("dida365_username")
		didaPassword := os.Getenv("dida365_password")
		client, err := ticktick.NewClient(didaUsername, didaPassword, "dida365")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error in creating dida365 client": err.Error()})
			return
		}
		// create task
		task, err := ticktick.NewTask(client, fmt.Sprintf("%v [%v]", emailContent.Subject, emailContent.From), taskDescription, time.Time{}, "")
		task.Tags = append(task.Tags, "email")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error in creating dida365 task": err.Error()})
			return
		}
		task, err = client.CreateTask(task)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error in creating dida365 task": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"task_id": task.Id})
	} else if appmode == "smtp" {
		// * use smtp instead
		smtpServerName := os.Getenv("smtp_server_name")
		smtpServerPort := os.Getenv("smtp_server_port")
		smtpUsername := os.Getenv("smtp_username")
		smtpPassword := os.Getenv("smtp_password")
		smtpFrom := os.Getenv("smtp_from")
		smtpTo := os.Getenv("smtp_to")

		auth := sasl.NewPlainClient("", smtpUsername, smtpPassword)

		// Connect to the server, authenticate, set the sender and recipient,
		// and send the email all in one step.
		to := []string{smtpTo}
		msg := strings.NewReader(fmt.Sprintf("To: %v\r\n", smtpTo) +
			fmt.Sprintf("Subject: %v [%v]\r\n", emailContent.Subject, emailContent.From) +
			"\r\n" +
			fmt.Sprintf("%v\r\n", taskDescription))

		err := smtp.SendMailTLS(fmt.Sprintf("%v:%v", smtpServerName, smtpServerPort), auth, smtpFrom, to, msg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error in sending email": err.Error()})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error in appmode": "appmode is not in .env"})
		return
	}
}
