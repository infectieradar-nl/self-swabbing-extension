package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/utils"
	"github.com/influenzanet/study-service/pkg/studyengine"
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
		codeCheckGroup.POST("/is-study-full", h.isStudyFullEventHandl)
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
	var req studyengine.ExternalEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error.Printf("error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceID := req.InstanceID
	if instanceID != h.instanceID {
		msg := fmt.Sprintf("unexpected instanceID: %s", req.InstanceID)
		logger.Error.Printf(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	codeSurveyItem, err := utils.FindSurveyItemResponse(req.Response.Responses, "CodeVal")
	if err != nil {
		logger.Debug.Printf("%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	codeQuestionResponse, err := utils.FindResponseSlot(codeSurveyItem.Response, "rg.cv.ic")
	if err != nil {
		logger.Debug.Printf("%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	codeValue := codeQuestionResponse.Value
	if codeValue == "" {
		logger.Error.Println("code value is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "code value is empty"})
		return
	}

	err = h.dbService.MarkEntryCodeAsUsed(h.instanceID, codeValue, req.ParticipantState.ParticipantID)
	if err != nil {
		logger.Error.Printf("%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "event processed successfully"})
}

func (h *HttpEndpoints) isStudyFullEventHandl(c *gin.Context) {
	var req studyengine.ExternalEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error.Printf("error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceID := req.InstanceID
	if instanceID != h.instanceID {
		msg := fmt.Sprintf("unexpected instanceID: %s", req.InstanceID)
		logger.Error.Printf(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// clean up unconfirmed reserved slots
	count, err := h.dbService.CountUsedCodes(instanceID)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusOK, gin.H{"value": false})
		return
	}
	logger.Debug.Printf("number of participants currently: %d", count)

	if count < h.samplerConfig.MaxNrOfParticipants {
		c.JSON(http.StatusOK, gin.H{"value": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"value": true})
}
