package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"regexp"

	// "net/url"
	"strconv"
	"time"
	"trash-separator/structs"

	"crypto/rand"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

func (idb *InDB) AuthLogin(c *gin.Context) {
	var (
		user      structs.User
		email     string
		password  string
		userToken string
		result    gin.H
	)

	tempUser := structs.User{}
	jsonData, err := ioutil.ReadAll(c.Request.Body)

	// email = c.PostForm("email")
	// password = c.PostForm("password")

	// if email == "" || password == "" {
	if err != nil {
		result = gin.H{"status": "error", "msg": "Failed parsing email and/or password"}
		c.JSON(http.StatusBadRequest, result)
		return
	}
	json.Unmarshal(jsonData, &tempUser)
	email = tempUser.Email
	password = tempUser.Password
	// log.Printf("email: %s, password: %s", email, password)

	resultQueryUser := idb.DB.Table("user").Select("*").Where("email = ?", email).First(&user)
	if resultQueryUser.Error != nil { //some error
		if resultQueryUser.Error == gorm.ErrRecordNotFound { //got no record from db
			result = gin.H{"status": "not authorized", "msg": "Invalid email and/or password combination"}
			c.JSON(http.StatusUnauthorized, result)
			return
		} else { //other db errors
			result = gin.H{"status": "error", "msg": fmt.Sprintf("Internal DB error, error: %v", resultQueryUser.Error.Error())}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil { //same password
		//create random token
		userToken = randToken()
		//put into redis, with expiry of 14 days. [key = token, value = user Struct]
		log.Printf("usertoken = %s", userToken)
		userJSON, err := json.Marshal(user)
		if err != nil {
			result = gin.H{"status": "error", "msg": fmt.Sprintf("Error parsing user into JSON, user: %v", user)}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		redisError := idb.RedisClient.Set(userToken, userJSON, time.Hour*24*14)
		if redisError.Err() != nil {
			result = gin.H{"status": "error", "msg": fmt.Sprintf("Internal Redis DB error, error: %v", redisError.Err().Error())}
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		//put into user cookie then redirect
		c.SetCookie("user_token", userToken, 3600*24*14, "/", c.Request.URL.Hostname(), false, true)
		c.SetCookie("user_id", strconv.Itoa(user.Id), 3600*24*14, "/", c.Request.URL.Hostname(), false, true)
		c.SetCookie("user_name", user.Name, 3600*24*14, "/", c.Request.URL.Hostname(), false, true)
		c.SetCookie("user_address", user.Address, 3600*24*14, "/", c.Request.URL.Hostname(), false, true)
		c.SetCookie("user_company", user.Company_name, 3600*24*14, "/", c.Request.URL.Hostname(), false, true)
		c.SetCookie("user_telephone", user.Telephone, 3600*24*14, "/", c.Request.URL.Hostname(), false, true)

		//redirection handled by frontend
		// location := url.URL{Path: "/"}
		// c.Redirect(http.StatusFound, location.RequestURI())
		result = gin.H{
			"status": "success",
			"msg":    "login successful",
		}
		c.JSON(http.StatusOK, result)

	} else { //wrong password
		result = gin.H{"status": "not authorized", "msg": "Invalid email and/or password combination"}
		c.JSON(http.StatusUnauthorized, result)
		return
	}
}

func (idb *InDB) AuthRegister(c *gin.Context) {
	var (
		userInput structs.User

		result gin.H
	)

	// userInput = structs.User{
	// 	Name:         c.PostForm("name"),
	// 	Email:        c.PostForm("email"),
	// 	Telephone:    c.PostForm("telephone"),
	// 	Address:      c.PostForm("address"),
	// 	Company_name: c.PostForm("company_name"),
	// 	Password:     c.PostForm("password"),
	// }
	// confirmPass = c.PostForm("confirm_password")

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		result = gin.H{"status": "error", "msg": "Failed parsing email and/or password"}
		c.JSON(http.StatusBadRequest, result)
		return
	}
	json.Unmarshal(jsonData, &userInput)
	// a := string(jsonData)
	// log.Printf("JSON: %s", a)
	// log.Printf("userInput password: %s", userInput.Password)

	phoneRegex := "^[0-9+]{8,13}$" //number only, with length of 8 - 13
	re := regexp.MustCompile(phoneRegex)
	_, errEmail := mail.ParseAddress(userInput.Email)
	//validate input
	if userInput.Name == "" || errEmail != nil || !re.MatchString(userInput.Telephone) || userInput.Address == "" || userInput.Company_name == "" || userInput.Password == "" {
		result = gin.H{"status": "error", "msg": "Invalid inputs!"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userInput.Password), bcrypt.DefaultCost)
	if err != nil {
		result = gin.H{"status": "error", "msg": "Password Hashing Failed!"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	userInput.Password = string(hashedPassword[:])

	//check if email already has account or no
	resultEmails, err := idb.DB.Table("user").Select("email").Rows()
	if err != nil {
		result = gin.H{"status": "error", "msg": fmt.Sprintf("internal db error: %v", err)}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	defer resultEmails.Close()
	for resultEmails.Next() {
		var dbEmail string
		resultEmails.Scan(&dbEmail)
		if dbEmail == userInput.Email {
			result = gin.H{"status": "error", "msg": "This email is already registered!"}
			c.JSON(http.StatusBadRequest, result)
			return
		}
	}
	//push to db
	userInput.Created_date = time.Now()
	userInput.Updated_date = time.Now()
	resultInsertUser := idb.DB.Table("user").Create(&userInput)

	if resultInsertUser.Error == nil {
		result = gin.H{"status": "success", "msg": "Register successful, please redirect user"}
		c.JSON(http.StatusOK, result)
		return
	} else {
		result = gin.H{"status": "error", "msg": fmt.Sprintf("internal db error in insert user, error: %v", resultInsertUser.Error.Error())}
		c.JSON(http.StatusInternalServerError, result)
		return
	}
}

func randToken() string {
	b := make([]byte, 16) //length is 16*2 characters
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (idb *InDB) NotImplemented(c *gin.Context) {
	res := gin.H{"status": "not implemented", "msg": "not implemented"}
	c.JSON(http.StatusOK, res)
}

func (idb *InDB) CheckAuth(c *gin.Context) {

	var res gin.H
	var err error
	var redisResponse string

	userToken, err := c.Cookie("user_token")
	if err != nil {
		res = gin.H{
			"status": "unauthorized",
		}
		c.JSON(http.StatusUnauthorized, res)
	}

	redisResponse, err = idb.RedisClient.Get(userToken).Result()
	if err != nil {
		res = gin.H{
			"status": "unauthorized",
		}
		c.JSON(http.StatusUnauthorized, res)
	}

	// Check if user is admin
	user := structs.User{}
	json.Unmarshal([]byte(redisResponse), &user)

	resultGetUser := idb.DB.Table("user").
		Select("admin").
		Where("id = ?", user.Id).
		First(&user)

	if resultGetUser.Error != nil {
		res = gin.H{
			"status": "unauthorized", //user doesn't exist
		}
		c.JSON(http.StatusUnauthorized, res)
	}

	res = gin.H{
		"status":  "logged in",
		"isAdmin": user.Admin,
	}
	c.JSON(http.StatusOK, res)
}

func (idb *InDB) AuthLogout(c *gin.Context) {
	// remove user_token from cookie
	token, err := c.Cookie("user_token")
	if err != nil {
		//cookie is not there
		res := gin.H{
			"status": "logged out",
			"msg":    "token not found",
		}
		c.JSON(http.StatusOK, res)
		return
	}
	c.SetCookie("user_token", "", 0, "/", c.Request.URL.Hostname(), false, true)
	c.SetCookie("user_id", "", 0, "/", c.Request.URL.Hostname(), false, true)
	c.SetCookie("user_name", "", 0, "/", c.Request.URL.Hostname(), false, true)
	c.SetCookie("user_address", "", 0, "/", c.Request.URL.Hostname(), false, true)
	c.SetCookie("user_company", "", 0, "/", c.Request.URL.Hostname(), false, true)
	c.SetCookie("user_telephone", "", 0, "/", c.Request.URL.Hostname(), false, true)

	// remove from redis
	redisRes := idb.RedisClient.Del(token)
	if redisRes.Err() != nil {
		//error deleting
		res := gin.H{
			"status": "error",
			"msg":    fmt.Sprintf("Redis database error: %v", redisRes.Err().Error()),
		}
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	res := gin.H{
		"status": "logged out",
		"msg":    "successfully logged out",
	}
	c.JSON(http.StatusOK, res)
}
