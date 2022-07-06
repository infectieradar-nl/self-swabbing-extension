package main

import (
	"net/http"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/db"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/http/handlers"
)

var conf Config

func init() {
	conf = initConfig()
	if !conf.GinDebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	logger.SetLevel(conf.LogLevel)
}

func healthCheckHandle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func main() {
	logger.Info.Println("Starting self-swabbing-extension")

	dbService := db.NewSelfSwabbingExtDBService(conf.DBConfig)

	// Start webserver
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		// AllowAllOrigins: true,
		AllowOrigins:     conf.AllowOrigins,
		AllowMethods:     []string{"POST", "GET", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Content-Length", "Api-Key"},
		ExposeHeaders:    []string{"Authorization", "Content-Type", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.GET("/", healthCheckHandle)
	apiRoot := router.Group("")

	apiHandlers := handlers.NewHTTPHandler(
		dbService,
		conf.APIKeys,
		conf.AllowEntryCodeUpload,
	)
	apiHandlers.AddCodeCheckerAPI(apiRoot)

	logger.Info.Printf("self swabbing extension is listening on port %s", conf.Port)
	logger.Error.Fatal(router.Run(":" + conf.Port))
}
