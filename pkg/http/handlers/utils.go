package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
)

func RecordBodyHandl(c *gin.Context) {
	req, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to read request body",
		})
		return
	}

	filename := fmt.Sprintf("%s.json", time.Now().Format("2006-01-02-15-04-05"))
	err = ioutil.WriteFile(filename, req, 0644)
	if err != nil {
		logger.Error.Println(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to save the file",
		})
		return
	}

	// File saved successfully. Return proper result
	c.JSON(http.StatusOK, gin.H{"message": "Your file has been successfully saved."})
}

func SanitizeCode(code string) string {
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "_", "")
	code = strings.ReplaceAll(code, "-", "")
	return code
}
