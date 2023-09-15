package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ziyixi/Cloudmailin-Todoist-Sync/src/handleRouter"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	// basic auth
	godotenv.Load() // it's OK if no .env, as we read from ENV variables instead
	cmiUser := os.Getenv("cloudmailin_username")
	cmiPass := os.Getenv("cloudmailin_password")
	if len(cmiUser) == 0 || len(cmiPass) == 0 {
		panic("cloudmailin_username or cloudmailin_password is not in .env")
	}

	// check other required env variables
	appmode := os.Getenv("appmode")
	todoistApiKey := os.Getenv("todoist_api_key")
	didaUsername := os.Getenv("dida365_username")
	didaPassword := os.Getenv("dida365_password")
	smtpServerName := os.Getenv("smtp_server_name")
	smtpServerPort := os.Getenv("smtp_server_port")
	smtpUsername := os.Getenv("smtp_username")
	smtpPassword := os.Getenv("smtp_password")
	smtpFrom := os.Getenv("smtp_from")
	smtpTo := os.Getenv("smtp_to")

	if appmode == "todoist" {
		if len(todoistApiKey) == 0 {
			panic("todoist_api_key is not in .env")
		}
	} else if appmode == "dida365" {
		if len(didaUsername) == 0 && len(didaPassword) == 0 {
			panic("dida365_username and dida365_password is not in .env")
		} else if len(didaUsername) == 0 {
			panic("dida365_username is not in .env")
		} else if len(didaPassword) == 0 {
			panic("dida365_password is not in .env")
		}
	} else if appmode == "smtp" {
		if len(smtpServerName) == 0 && len(smtpServerPort) == 0 && len(smtpUsername) == 0 && len(smtpPassword) == 0 && len(smtpFrom) == 0 && len(smtpTo) == 0 {
			panic("smtp_server_name, smtp_server_port, smtp_username, smtp_password, smtp_from, smtp_to is not in .env")
		}
	} else {
		panic("appmode is not in .env")
	}

	openaiApiKey := os.Getenv("openai_api_key")
	if len(openaiApiKey) == 0 {
		panic("openai_api_key is not in .env")
	}

	// routes
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		cmiUser: cmiPass,
	}))
	authorized.POST("/api/CloudmailinTodoistSync", handleRouter.HandleCloudmailinPost)

	return r
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	listenAddr := ":" + os.Getenv("PORT")

	r := setupRouter()
	r.Run(listenAddr)
}
