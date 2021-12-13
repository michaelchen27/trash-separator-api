package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"trash-separator/structs"
	"trash-separator/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (idb *InDB) SendLog(c *gin.Context) {
	var (
		result gin.H
		status string
		msg    string
	)

	trashcanID, err := strconv.Atoi(c.PostForm("trash_can_id"))
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid trash id format"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	category := c.PostForm("category")
	if category != "inorganic" && category != "organic" {
		result = gin.H{"status": "error", "msg": "invalid trash category"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	trashType := c.PostForm("type")

	fmt.Println("Trash Type:", trashType)

	insertLog := structs.Trash_reading{
		Trash_id:     trashcanID,
		Category:     category,
		Type:         trashType,
		Created_date: time.Now(),
	}

	resultInsertLog := idb.DB.Table("trash_reading").Create(&insertLog)

	if resultInsertLog.Error == nil {
		status = "success"
		msg = "Log successfully added"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusOK, result)

	} else {
		fmt.Println("ERROR: ", resultInsertLog.Error)

		status = "error"
		msg = "Log insertion failed"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusInternalServerError, result)

	}
}

func (idb *InDB) SendCapacity(c *gin.Context) {
	var (
		result gin.H
		status string
		msg    string
	)

	trashcanID, _ := strconv.Atoi(c.PostForm("trash_can_id"))
	organicCapacity, _ := strconv.Atoi(c.PostForm("organic_capacity"))
	inorganicCapacity, _ := strconv.Atoi(c.PostForm("inorganic_capacity"))

	insertCapacity := structs.Trash_capacity{
		Trash_id:           trashcanID,
		Organic_capacity:   organicCapacity,
		Inorganic_capacity: inorganicCapacity,
		Created_at:         time.Now(),
	}

	resultInsertCapacity := idb.DB.Table("trash_capacity").Create(&insertCapacity)

	if resultInsertCapacity.Error == nil {
		status = "success"
		msg = "Capacity log successfully added"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusOK, result)
	} else {
		fmt.Println("ERROR: ", resultInsertCapacity.Error)

		status = "error"
		msg = "Capacity Log insertion failed"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusInternalServerError, result)
	}
}

func (idb *InDB) GetSingleTrashCanCapacity(c *gin.Context) {
	var (
		trashCapacity structs.TrashCapacity
		status        string
		msg           string
		result        gin.H
	)

	trashCanID := c.Param("trash_can_id")

	resultGetSingleTrashCapacity := idb.DB.Table("trash_capacity").
		Select("trash_capacity.*, trash_version.organic_max_height, trash_version.inorganic_max_height").
		Joins("left join trash on trash.id = trash_capacity.trash_id").
		Joins("left join trash_version on trash_version.id = trash.trash_version_id").
		Where("trash_capacity.trash_id = ?", trashCanID).
		Last(&trashCapacity)

	if resultGetSingleTrashCapacity.Error == gorm.ErrRecordNotFound { //attempt to get trash
		resultTrash, err := idb.DB.Table("trash").
			Select("trash_version.organic_max_height, trash_version.inorganic_max_height").
			Joins("left join trash_version on trash_version.id = trash.trash_version_id").
			Where("trash.id = ?", trashCanID).
			Rows()
		trashCapacity.Trash_id, _ = strconv.Atoi(trashCanID)
		if err != nil {
			status = "error"
			msg = err.Error()

			result = gin.H{
				"status": status,
				"msg":    msg,
			}

			c.JSON(http.StatusInternalServerError, result)
			return
		}
		defer resultTrash.Close()
		for resultTrash.Next() {
			var organic_max, inorganic_max int
			errRow := resultTrash.Scan(&organic_max, &inorganic_max)
			if errRow != nil {
				result = gin.H{
					"status": "error",
					"msg":    err.Error(),
				}
				c.JSON(http.StatusInternalServerError, result)
				return
			}
			trashCapacity.Organic_max_height = organic_max
			trashCapacity.Inorganic_max_height = inorganic_max
		}
		status = "success"
		if trashCapacity.Organic_max_height == 0 && trashCapacity.Inorganic_max_height == 0 {
			msg = fmt.Sprintf("Trash with ID : " + trashCanID + " not found")
		} else {
			msg = "Get single trash can capacity successful (empty capacity)"
		}
		result = gin.H{
			"status": status,
			"msg":    msg,
			"data":   trashCapacity,
		}
		c.JSON(http.StatusOK, result)
		return

	} else if resultGetSingleTrashCapacity.Error == nil {
		status = "success"
		msg = "Get single trash can capacity successful"

		result = gin.H{
			"status": status,
			"msg":    msg,
			"data":   trashCapacity,
		}

		c.JSON(http.StatusOK, result)

	} else {
		status = "error"
		msg = resultGetSingleTrashCapacity.Error.Error()

		result = gin.H{
			"status": status,
			"msg":    msg,
		}

		c.JSON(http.StatusInternalServerError, result)

	}
}

func (idb *InDB) GetAllTrashCanLogs(c *gin.Context) {
	var (
		trashLogs     []structs.TrashLogs
		trashReadings map[int][]structs.Trash_reading
		status        string
		msg           string
		result        gin.H
	)

	trashReadings = map[int][]structs.Trash_reading{}
	rows, err := idb.DB.Table("trash_reading").Select("*").Rows()
	if idb.DB.Error == gorm.ErrRecordNotFound {
		result = gin.H{"status": "success", "msg": "no data"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reading structs.Trash_reading
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		trashReadings[reading.Trash_id] = append(trashReadings[reading.Trash_id], reading)
	}

	for key, val := range trashReadings {
		var temp structs.TrashLogs
		temp.Trash_can_id = key
		temp.Trash_reading = val
		trashLogs = append(trashLogs, temp)
	}

	status = "fetch log ok"
	msg = "ok"
	if len(trashLogs) == 0 {
		msg = "empty logs"
	}
	result = gin.H{
		"status": status,
		"msg":    msg,
		"data":   trashLogs,
	}

	c.JSON(http.StatusOK, result)

}

func (idb *InDB) GetSingleTrashCanLogs(c *gin.Context) {
	var (
		trashLogs        structs.TrashLogs
		trashReadings    []structs.Trash_reading
		trashCanIDString string
		trashCanID       int
		status           string
		msg              string
		result           gin.H
		err              error
	)
	trashCanIDString = c.Param("trash_can_id")
	trashCanID, err = strconv.Atoi(trashCanIDString)
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid trash id format"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	trashReadings = []structs.Trash_reading{}
	rows, err := idb.DB.Table("trash_reading").Select("*").Where("trash_id = ?", trashCanID).Rows()
	if idb.DB.Error == gorm.ErrRecordNotFound {
		result = gin.H{"status": "success", "msg": "no data"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reading structs.Trash_reading
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		trashReadings = append(trashReadings, reading)
	}

	trashLogs.Trash_can_id = trashCanID
	trashLogs.Trash_reading = trashReadings

	status = "fetch log ok"
	msg = "ok"
	if len(trashLogs.Trash_reading) == 0 {
		msg = "empty logs"
	}
	result = gin.H{
		"status": status,
		"msg":    msg,
		"data":   trashLogs,
	}

	c.JSON(http.StatusOK, result)

}

func (idb *InDB) GetTopTrashCans(c *gin.Context) {
	var (
		trashReading []structs.TrashReading
		status       string
		msg          string
		userId       string
		result       gin.H
	)

	tn := time.Now()
	year, week := tn.ISOWeek()
	firstDayOfWeek := util.WeekStart(year, week)
	userId = getUserIdFromRedis(idb, c)

	log.Printf("User id : %s", userId)
	resultGetTopTrashCans := idb.DB.Table("trash_reading").
		Select("trash.trash_code AS Trash_sorter_name, trash.location AS Trash_sorter_location, count(*) as Total").
		Joins("left join trash on trash.id = trash_reading.trash_id").
		Group("trash.id").
		Where("trash_reading.created_date BETWEEN ? AND ?", firstDayOfWeek, tn).
		Where("trash.user_id = ?", userId).
		Order("Total desc").
		Limit(5).
		Find(&trashReading)

	if resultGetTopTrashCans.Error != nil {
		status = "error"
		msg = fmt.Sprintf("db error: %v", resultGetTopTrashCans.Error.Error())
		result = gin.H{
			"status": status,
			"msg":    msg,
		}
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	if len(trashReading) == 0 { //empty reading, try to fetch trash name only
		resultTrashName := idb.DB.Table("trash").
			Select("trash_code AS Trash_sorter_name, location AS Trash_sorter_location").
			Where("user_id = ?", userId).
			Limit(5).Find(&trashReading)
		if resultTrashName.Error != nil {
			result = gin.H{
				"status": "error",
				"msg":    fmt.Sprintf("Error trying to get trash name, Db error: %v", resultTrashName.Error.Error()),
			}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		if len(trashReading) == 0 { // user has no trash can
			trashReading = append(trashReading, structs.TrashReading{})
			msg = "There is no trash can associated with the user!"
		} else {
			msg = "No record in current week!"
		}
		result = gin.H{
			"user_id": userId,
			"data":    trashReading,
			"status":  "success",
			"msg":     msg,
		}
		c.JSON(http.StatusOK, result)
		return
	}

	if len(trashReading) != 0 {
		status = "success"
		msg = "Successfully get current week data"

		result = gin.H{
			"user_id": userId,
			"data":    trashReading,
			"status":  status,
			"msg":     msg,
		}

		c.JSON(http.StatusOK, result)
	}

}

func (idb *InDB) GetTrashSummaryWeek(c *gin.Context) {
	var (
		resp             structs.SummaryResponse
		trashCanID       int
		trashCanIDString string
		msg              string
		status           string
		err              error
		result           gin.H
	)
	resp = structs.SummaryResponse{
		Type:     make(map[string]int),
		Category: make(map[string]int),
	}
	resp.Category["inorganic"] = 0
	resp.Category["organic"] = 0

	trashCanIDString = c.Param("trash_can_id")
	trashCanID, err = strconv.Atoi(trashCanIDString)
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid trash id format"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	tn := time.Now()
	tnPrevWeek := tn.AddDate(0, 0, -7)

	rows, err := idb.DB.Table("trash_reading").
		Select("*").
		Where("created_date BETWEEN ? AND ?", tnPrevWeek, tn).
		Where("trash_id = ?", trashCanID).Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reading structs.Trash_reading
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		resp.Category[reading.Category]++
		resp.Type[reading.Type]++
	}
	status = "fetch summary ok"
	msg = "ok"
	if resp.Category["inorganic"]+resp.Category["organic"] == 0 {
		msg = "empty logs"
	}
	result = gin.H{
		"status": status,
		"msg":    msg,
		"data":   resp,
	}

	c.JSON(http.StatusOK, result)
}

func (idb *InDB) GetTrashTypeAllByUserSummary(c *gin.Context) {
	var (
		resp   structs.SummaryResponse
		userId string
		msg    string
		status string
		err    error
		result gin.H
	)
	resp = structs.SummaryResponse{
		Type:     make(map[string]int),
		Category: make(map[string]int),
	}
	resp.Category["inorganic"] = 0
	resp.Category["organic"] = 0

	userId = getUserIdFromRedis(idb, c)
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid trash id format"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	tn := time.Now()
	tnPrevWeek := tn.AddDate(0, 0, -7)

	rows, err := idb.DB.Table("trash_reading").
		Select("trash_reading.*").
		Joins("left join trash on trash.id = trash_reading.trash_id").
		Where("trash_reading.created_date BETWEEN ? AND ?", tnPrevWeek, tn).
		Where("trash.user_id = ?", userId).Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reading structs.Trash_reading
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		log.Printf(reading.Category)
		resp.Category[reading.Category]++
		resp.Type[reading.Type]++
	}
	status = "fetch summary ok"
	msg = "ok"
	if resp.Category["inorganic"]+resp.Category["organic"] == 0 {
		msg = "empty logs"
	}
	result = gin.H{
		"status": status,
		"msg":    msg,
		"data":   resp,
	}

	c.JSON(http.StatusOK, result)
}

func (idb *InDB) GetTrashTypeWeek(c *gin.Context) {
	var (
		chartData        map[time.Time]map[string]int
		resp             []structs.TypeChartResponse
		trashCanID       int
		trashCanIDString string
		msg              string
		status           string
		err              error
		result           gin.H
	)
	chartData = make(map[time.Time]map[string]int)

	trashCanIDString = c.Param("trash_can_id")
	trashCanID, err = strconv.Atoi(trashCanIDString)
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid trash id format"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	tn := time.Now()
	rounded := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour()+1, 0, 0, 0, tn.Location())
	rPrevWeek := rounded.AddDate(0, 0, -7)

	rows, err := idb.DB.Table("trash_reading").
		Select("*").
		Where("created_date BETWEEN ? AND ?", rPrevWeek, rounded).
		Where("trash_id = ?", trashCanID).Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			reading         structs.Trash_reading
			roundedNextHour time.Time
			t               time.Time
		)
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		t = reading.Created_date
		roundedNextHour = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
		if chartData[roundedNextHour] == nil {
			chartData[roundedNextHour] = make(map[string]int)
		}
		chartData[roundedNextHour][reading.Type]++
	}

	//create response
	resp = []structs.TypeChartResponse{}
	types := make(map[string]bool)
	//fill missing hours
	for rPrevWeek.Before(tn) {
		tempResp := structs.TypeChartResponse{}
		if chartData[rPrevWeek] == nil {
			chartData[rPrevWeek] = make(map[string]int)
		}
		tempResp.Created_date = rPrevWeek
		for key, val := range chartData[rPrevWeek] {
			temp := make(map[string]string)
			temp["name"] = key
			temp["value"] = strconv.Itoa(val)
			types[key] = true
			tempResp.Data_type = append(tempResp.Data_type, temp)
		}
		resp = append(resp, tempResp)

		rPrevWeek = rPrevWeek.Add(time.Hour)
	}
	var typeArr []string
	for key := range types {
		typeArr = append(typeArr, key)
	}

	status = "fetch chart ok"
	msg = "ok"
	result = gin.H{
		"status":         status,
		"msg":            msg,
		"data":           resp,
		"type_available": typeArr,
	}

	c.JSON(http.StatusOK, result)
}

func (idb *InDB) GetTrashTypeAllByUser(c *gin.Context) {
	var (
		chartData map[time.Time]map[string]int
		resp      []structs.TypeChartResponse
		userId    string
		msg       string
		status    string
		err       error
		result    gin.H
	)
	chartData = make(map[time.Time]map[string]int)

	userId = getUserIdFromRedis(idb, c)

	tn := time.Now()
	rounded := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour()+1, 0, 0, 0, tn.Location())
	rPrevWeek := rounded.AddDate(0, 0, -7)

	rows, err := idb.DB.Table("trash_reading").
		Select("trash_reading.*").
		Joins("left join trash on trash.id = trash_reading.trash_id").
		Where("trash_reading.created_date BETWEEN ? AND ?", rPrevWeek, rounded).
		Where("trash.user_id = ?", userId).Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": fmt.Sprintf("internal db error: %v", err)}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			reading         structs.Trash_reading
			roundedNextHour time.Time
			t               time.Time
		)
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		t = reading.Created_date
		roundedNextHour = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
		if chartData[roundedNextHour] == nil {
			chartData[roundedNextHour] = make(map[string]int)
		}
		chartData[roundedNextHour][reading.Type]++
	}

	//create response
	resp = []structs.TypeChartResponse{}
	types := make(map[string]bool)
	//fill missing hours
	for rPrevWeek.Before(tn) {
		tempResp := structs.TypeChartResponse{}
		if chartData[rPrevWeek] == nil {
			chartData[rPrevWeek] = make(map[string]int)
		}
		tempResp.Created_date = rPrevWeek
		for key, val := range chartData[rPrevWeek] {
			temp := make(map[string]string)
			temp["name"] = key
			temp["value"] = strconv.Itoa(val)
			types[key] = true
			tempResp.Data_type = append(tempResp.Data_type, temp)
		}
		resp = append(resp, tempResp)

		rPrevWeek = rPrevWeek.Add(time.Hour)
	}
	var typeArr []string
	for key := range types {
		typeArr = append(typeArr, key)
	}

	status = "fetch chart by user ok"
	msg = "ok"
	result = gin.H{
		"status":         status,
		"msg":            msg,
		"data":           resp,
		"type_available": typeArr,
	}

	c.JSON(http.StatusOK, result)
}

func (idb *InDB) GetAllTrashCanByUser(c *gin.Context) {
	var (
		trashCans []structs.Trash
		status    string
		msg       string
		userId    string
		result    gin.H
	)

	userId = getUserIdFromRedis(idb, c)

	rows, err := idb.DB.Table("trash").
		Select("*").
		Where("user_id = ?", userId).Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": "internal db error"}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var reading structs.Trash
		idb.DB.ScanRows(rows, &reading)

		if idb.DB.Error != nil {
			result = gin.H{"status": "error", "msg": "internal db error"}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		trashCans = append(trashCans, reading)
	}

	status = "fetch log ok"
	msg = "ok"
	if len(trashCans) == 0 {
		msg = "empty logs"
	}
	result = gin.H{
		"status": status,
		"msg":    msg,
		"data":   trashCans,
	}

	c.JSON(http.StatusOK, result)
}

func getUserIdFromRedis(idb *InDB, c *gin.Context) string {
	userToken, _ := c.Cookie("user_token")
	redisResp, err := idb.RedisClient.Get(userToken).Result()
	if err != nil {
		return "2" //placeholder
	}
	user := structs.User{}
	json.Unmarshal([]byte(redisResp), &user)
	return strconv.Itoa(user.Id)
}

func (idb *InDB) GetAllTrashVersion(c *gin.Context) {
	var (
		trashVersion []structs.Trash_version
		result       gin.H
		status       string
		msg          string
	)

	resultGetAllTrashVersion := idb.DB.Table("trash_version").Find(&trashVersion)

	if resultGetAllTrashVersion.Error == nil {
		status = "success"
		msg = "Successfully get all data of trash version"

		result = gin.H{
			"status": status,
			"msg":    msg,
			"count":  resultGetAllTrashVersion.RowsAffected,
			"data":   trashVersion,
		}

		c.JSON(http.StatusOK, result)

	} else {
		status = "error"
		msg = resultGetAllTrashVersion.Error.Error()

		result = gin.H{
			"status": status,
			"msg":    msg,
		}
		c.JSON(http.StatusInternalServerError, result)

	}
}

func (idb *InDB) AddTrashVersion(c *gin.Context) {
	var (
		result gin.H
		status string
		msg    string
	)

	versionName := c.PostForm("version_name")
	if versionName == "" {
		result = gin.H{"status": "error", "msg": "invalid trash version name (empty string)"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	organicMaxHeight, err := strconv.Atoi(c.PostForm("organic_max"))
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid max value"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	inorganicMaxHeight, err := strconv.Atoi(c.PostForm("inorganic_max"))
	if err != nil {
		result = gin.H{"status": "error", "msg": "invalid max value"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	insertTrashVersion := structs.Trash_version{
		Version_name:         versionName,
		Inorganic_max_height: inorganicMaxHeight,
		Organic_max_height:   organicMaxHeight,
	}

	resultInsertTrashVersion := idb.DB.Table("trash_version").Create(&insertTrashVersion)

	if resultInsertTrashVersion.Error == nil {
		status = "success"
		msg = "Trash version successfully added"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusOK, result)

	} else {
		status = "error"
		msg = "Trash version insertion failed"

		result = gin.H{
			"status":  status,
			"message": msg,
		}

		c.JSON(http.StatusInternalServerError, result)
	}

}

func (idb *InDB) EditTrashVersion(c *gin.Context) {
	var (
		trashVersion structs.Trash_version
		status       string
		msg          string
		result       gin.H
	)

	trashVersionID := c.Param("trash_version_id")

	resultGetTrashVersion := idb.DB.Table("trash_version").
		Where("trash_version.id = ?", trashVersionID).
		First(&trashVersion)

	if resultGetTrashVersion.Error == nil {

		versionName := c.PostForm("version_name")
		if versionName == "" {
			result = gin.H{"status": "error", "msg": "invalid trash version name (empty string)"}
			c.JSON(http.StatusBadRequest, result)
			return
		}

		organicMax, err := strconv.Atoi(c.PostForm("organic_max"))
		if err != nil {
			result = gin.H{"status": "error", "msg": "invalid organic max value"}
			c.JSON(http.StatusBadRequest, result)
			return
		}

		inorganicMax, err := strconv.Atoi(c.PostForm("inorganic_max"))
		if err != nil {
			result = gin.H{"status": "error", "msg": "invalid inorganic max value"}
			c.JSON(http.StatusBadRequest, result)
			return
		}

		updateTrashVersion := structs.Trash_version{
			Version_name:         versionName,
			Organic_max_height:   organicMax,
			Inorganic_max_height: inorganicMax,
		}

		resultUpdateTrashVersion := idb.DB.Model(&trashVersion).Updates(&updateTrashVersion)

		if resultUpdateTrashVersion.Error == nil {
			status = "success"
			msg = "TrashVersionID " + trashVersionID + ", successfully updated"

			result = gin.H{
				"status": status,
				"msg":    msg,
			}

			c.JSON(http.StatusOK, result)
		} else {
			status = "error"
			msg = "Trash version update failed"
			fmt.Println(resultUpdateTrashVersion.Error.Error())

			result = gin.H{
				"status": status,
				"msg":    msg,
			}

			c.JSON(http.StatusInternalServerError, result)

		}

	} else {
		status = "error"
		// msg = resultGetTrashVersion.Error.Error()
		msg = "Trash version doesn't exist"

		result = gin.H{
			"status": status,
			"msg":    msg,
		}

		c.JSON(http.StatusInternalServerError, result)

	}
}

func (idb *InDB) DeleteTrashVersion(c *gin.Context) {
	var (
		trashVersion structs.Trash_version
		status       string
		msg          string
		result       gin.H
	)

	trashVersionID := c.Param("trash_version_id")

	resultDeleteTrashVersion := idb.DB.Delete(&trashVersion, trashVersionID)

	if resultDeleteTrashVersion.Error == nil {
		status = "success"
		msg = "Trash Version ID " + trashVersionID + ", successfully deleted"

		result = gin.H{
			"status": status,
			"msg":    msg,
		}

		c.JSON(http.StatusOK, result)

	} else {
		status = "error"
		msg = "Trash version deletion failed"
		fmt.Println(resultDeleteTrashVersion.Error.Error())

		result = gin.H{
			"status": status,
			"msg":    msg,
		}

		c.JSON(http.StatusInternalServerError, result)
	}

}
