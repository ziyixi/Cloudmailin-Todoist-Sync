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
	todoistApiKey := os.Getenv("todoist_api_key")
	didaUsername := os.Getenv("dida365_username")
	didaPassword := os.Getenv("dida365_password")
	if len(todoistApiKey) == 0 {
		if len(didaUsername) == 0 && len(didaPassword) == 0 {
			panic("todoist_api_key or dida365_username and dida365_password is not in .env")
		} else if len(didaUsername) == 0 {
			panic("dida365_username is not in .env")
		} else if len(didaPassword) == 0 {
			panic("dida365_password is not in .env")
		}
		// else just use dida365
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
	// gin.SetMode(gin.ReleaseMode)

	listenAddr := ":" + os.Getenv("PORT")

	r := setupRouter()
	r.Run(listenAddr)
}
