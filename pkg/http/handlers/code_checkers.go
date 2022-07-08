package handlers

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
)

const (
	wrongCodeAttemptLimit                    = 10
	wrongCodeAttemptLimitWindowResetInterval = 5 * 60
	randomDelayMax                           = 10
)

var (
	wrongCodeChecksPerUID map[string]int
	lastReset             int64
)

func (h *HttpEndpoints) AddCodeCheckerAPI(rg *gin.RouterGroup) {
	codeCheckGroup := rg.Group("/entry-codes/:instanceID")
	codeCheckGroup.Use(mw.HasValidInstanceID())
	codeCheckGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{
		if h.allowEntryCodeUpload {
			codeCheckGroup.POST("", mw.RequirePayload(), h.addNewEntryCodesHandl)
		}
		codeCheckGroup.GET("/is-valid", h.validateEntryCodeHandl)
		codeCheckGroup.POST("/submit", mw.RequirePayload(), h.studyEventWithEntryCodeHandl)
	}

}

func (h *HttpEndpoints) addNewEntryCodesHandl(c *gin.Context) {
	instanceID := c.Param("instanceID")

	var req types.NewCodeList
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dbService.CreateIndexForEntryCodes(instanceID)
	if err != nil {
		logger.Error.Printf("unexpected error when creating index: %v", err)
	}

	counter := 0
	for _, c := range req.Codes {
		_, err := h.dbService.AddEntryCode(instanceID, c)
		if err != nil {
			logger.Error.Printf("unexpected error when saving entry code '%s': %v", c, err)
		} else {
			counter += 1
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d / %d codes saved", counter, len(req.Codes))})
}

func (h *HttpEndpoints) validateEntryCodeHandl(c *gin.Context) {
	instanceID := c.Param("instanceID")

	uid := c.DefaultQuery("uid", "")
	if uid == "" || len(uid) != 24 {
		logger.Warning.Println("empty uid when checking entry code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong id"})
		time.Sleep(time.Duration(rand.Intn(randomDelayMax)) * time.Second)
		return
	}

	code := c.DefaultQuery("code", "")
	if code == "" {
		logger.Warning.Println("empty entry code attempt")
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty entry code attempt"})
		time.Sleep(time.Duration(rand.Intn(randomDelayMax)) * time.Second)
		return
	}
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "_", "")
	code = strings.ReplaceAll(code, "-", "")

	now := time.Now().Unix()
	if now-lastReset > wrongCodeAttemptLimitWindowResetInterval {
		wrongCodeChecksPerUID = map[string]int{}
		lastReset = now
	}

	count, ok := wrongCodeChecksPerUID[uid]
	if ok && count > wrongCodeAttemptLimit {
		logger.Warning.Printf("%s too many wrong code attempts", uid)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong entry code"})
		time.Sleep(time.Duration(rand.Intn(randomDelayMax)) * time.Second)
		return
	}

	codeInfos, err := h.dbService.FindEntryCodeInfo(instanceID, code)
	if err != nil {
		if !ok {
			wrongCodeChecksPerUID[uid] = 1
		} else {
			wrongCodeChecksPerUID[uid] += 1
		}
		logger.Error.Printf("error when looking up code infos for '%s': %v", code, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong entry code"})
		time.Sleep(time.Duration(rand.Intn(randomDelayMax)) * time.Second)
		return
	}

	if codeInfos.UsedAt > 0 {
		if !ok {
			wrongCodeChecksPerUID[uid] = 1
		} else {
			wrongCodeChecksPerUID[uid] += 1
		}
		logger.Error.Printf("attempt to use expired code '%s': %v", code, codeInfos)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong entry code"})
		time.Sleep(time.Duration(rand.Intn(randomDelayMax)) * time.Second)
		return
	}

	c.JSON(http.StatusOK, gin.H{"isValid": true})
}

func (h *HttpEndpoints) studyEventWithEntryCodeHandl(c *gin.Context) {
	// TODO: receive and parse study event
	// TODO: find survey item and response item with code
	// TODO: update code in DB that is was used by participant

	resp, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to read request body",
		})
		return
	}

	err1 := ioutil.WriteFile("study_event.json", resp, 0644)

	if err1 != nil {
		fmt.Println("error:", err1)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to save the file",
		})
		return
	}

	// File saved successfully. Return proper result
	c.JSON(http.StatusOK, gin.H{
		"message": "Your file has been successfully saved."})
}
