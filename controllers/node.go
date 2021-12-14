package controllers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
	"trash-separator/structs"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (idb *InDB) RegisterUserTrashCan(c *gin.Context) {
	var (
		result                 gin.H
		trashObject, userInput structs.Trash
		timeNow                = time.Now()
	)

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		result = gin.H{"status": "error", "msg": "failed parsing json input"}
		c.JSON(http.StatusBadRequest, result)
		return
	}
	json.Unmarshal(jsonData, &userInput)
	if userInput.Trash_code == "" {
		result = gin.H{"status": "error", "msg": "failed, empty trash_code!"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	query := idb.DB.Table("trash").Select("*").Where("trash_code = ?", userInput.Trash_code).First(&trashObject)
	if query.Error != nil {
		log.Printf("error fetching data from database: %v", query.Error.Error())
		if query.Error == gorm.ErrRecordNotFound {
			result = gin.H{"status": "unauthorized", "mgs": "record not found"}
			c.JSON(http.StatusBadRequest, result)
		} else {
			result = gin.H{"status": "internal server error", "msg": "failed, database error fetch trash info"}
			c.JSON(http.StatusInternalServerError, result)
		}
		return
	}

	if trashObject.Assigned == "1" { //already assigned
		result = gin.H{"status": "unauthorized", "msg": "failed, trash object is already assigned to an user"}
		c.JSON(http.StatusUnauthorized, result)
		return
	}

	userId, err := c.Cookie("user_id")
	if err != nil {
		result = gin.H{"status": "unauthorized", "msg": "failed to get user_id from cookie"}
		c.JSON(http.StatusUnauthorized, result)
		return
	}

	trashObject.Assigned = "1"
	trashObject.Assigned_date = timeNow
	trashObject.Guarantee_expired_date = timeNow.Add(time.Hour * 24 * 365)
	trashObject.User_id, _ = strconv.Atoi(userId)

	if userInput.Location != "" {
		trashObject.Location = userInput.Location
	}
	if userInput.Custom_name != "" {
		trashObject.Custom_name = userInput.Custom_name
	}
	// trashObject.Latitude = userInput.Latitude
	// trashObject.Longitude = userInput.Longitude

	queryUpdate := idb.DB.Table("trash").Save(&trashObject)
	if queryUpdate.Error != nil {
		result = gin.H{"status": "internal server error", "msg": "failed, internal server error in inserting data to database"}
		log.Printf("error inserting to database: %v", queryUpdate.Error.Error())
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result = gin.H{"status": "success", "msg": "Register trash to user success"}
	c.JSON(http.StatusOK, result)
}
