package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"trash-separator/structs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func (idb *InDB) EnableMiddleware() {
	idb.Middleware = true
}

func (idb *InDB) MWCheckUserTokenCookie() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {

		if idb.Middleware {
			log.Println("Executing cookie middleware")
			//check for cookie
			user := structs.User{}
			token, err := c.Cookie("user_token")
			if err != nil {
				result := gin.H{"status": "not authorized", "msg": "You are not logged in"}
				c.JSON(http.StatusUnauthorized, result)
				c.Abort()
				return
			}
			log.Printf("token = %s", token)

			//check if cookie is valid or not
			resJSON, err := idb.RedisClient.Get(token).Result()
			if err == redis.Nil {
				//not found, delete all cookie
				c.SetCookie("user_token", "", 0, "/", c.Request.URL.Hostname(), false, true)
				c.SetCookie("user_id", "", 0, "/", c.Request.URL.Hostname(), false, true)
				c.SetCookie("user_name", "", 0, "/", c.Request.URL.Hostname(), false, true)
				c.SetCookie("user_address", "", 0, "/", c.Request.URL.Hostname(), false, true)
				c.SetCookie("user_company", "", 0, "/", c.Request.URL.Hostname(), false, true)
				c.SetCookie("user_telephone", "", 0, "/", c.Request.URL.Hostname(), false, true)

				result := gin.H{"status": "not authorized", "msg": "Please login again"}
				c.JSON(http.StatusUnauthorized, result)
				c.Abort()
				return
			}
			json.Unmarshal([]byte(resJSON), &user)
			log.Printf("resJSON = %s", resJSON)

		}

		c.Next()

	})
}

func (idb *InDB) MWCheckNodeToken() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {

		if idb.Middleware {
			log.Println("[NODE] Executing Node Token middleware")
			//check for cookie
			trashNode := structs.Trash{}
			token, err := c.Cookie("node_token")
			if err != nil {
				result := gin.H{"status": "not authorized", "msg": "invalid trash code"}
				c.JSON(http.StatusUnauthorized, result)
				c.Abort()
				return
			}
			log.Printf("[NODE] token = %s", token)

			//check if cookie is valid or not
			resJSON, err := idb.RedisClient.Get(token).Result()
			if err != nil && err != redis.Nil {
				log.Printf("Error retrieving trash info from redis, trash_code: %v", token)
				result := gin.H{"status": "internal server error", "msg": "error retrieving trash info from redis"}
				c.JSON(http.StatusInternalServerError, result)
				c.Abort()
				return
			}
			if err == redis.Nil {
				// check db for info
				resultTrash := idb.DB.Table("trash").Select("*").Where("trash_code = ?", token).First(&trashNode)
				if resultTrash.Error != nil {
					result := gin.H{"status": "not authorized", "msg": "Please login again"}
					c.JSON(http.StatusUnauthorized, result)
					c.Abort()
					return
				}

				// store token info in redis
				trashJSON, err := json.Marshal(trashNode)
				if err != nil {
					result := gin.H{"status": "internal server error", "msg": fmt.Sprintf("error marshal JSON, err: %v", err)}
					c.JSON(http.StatusUnauthorized, result)
					c.Abort()
					return
				}
				redisError := idb.RedisClient.Set(token, trashJSON, 0) //never expires
				if redisError.Err() != nil {
					log.Printf("Error setting to redis, err: %v", redisError.Err().Error())
				}
			} else {
				json.Unmarshal([]byte(resJSON), &trashNode)
			}

			log.Printf("trash Node info = %s", resJSON)

		}

		c.Next()

	})
}
