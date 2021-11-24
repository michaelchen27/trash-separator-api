package main

import (
	"os"
	"trash-separator/config"
	"trash-separator/controllers"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.DBInit()
	client := config.RedisInit()

	inDB := &controllers.InDB{DB: db, RedisClient: client}
	inDB.EnableMiddleware()

	router := gin.Default()

	// Microcontroller -> Server
	router.POST("/node/sendLog", inDB.SendLog)
	router.POST("/node/sendCapacity", inDB.SendCapacity)

	// Server -> Web
	router.GET("/node/getCapacity/:trash_can_id", inDB.MWCheckUserTokenCookie(), inDB.GetSingleTrashCanCapacity)
	router.GET("/node/getTopTrashCans", inDB.MWCheckUserTokenCookie(), inDB.GetTopTrashCans)

	router.GET("/node/getLogs/", inDB.MWCheckUserTokenCookie(), inDB.GetAllTrashCanLogs)
	router.GET("/node/getLogs/:trash_can_id", inDB.MWCheckUserTokenCookie(), inDB.GetSingleTrashCanLogs)

	// charts
	router.GET("/node/getWeeklySummary/:trash_can_id", inDB.MWCheckUserTokenCookie(), inDB.GetTrashSummaryWeek)
	router.GET("/node/getWeeklyTypes/:trash_can_id", inDB.MWCheckUserTokenCookie(), inDB.GetTrashTypeWeek)
	router.GET("/node/getAllTypesUser/", inDB.MWCheckUserTokenCookie(), inDB.GetTrashTypeAllByUser)
	router.GET("/node/getAllTypesUserSummary/", inDB.MWCheckUserTokenCookie(), inDB.GetTrashTypeAllByUserSummary)

	router.GET("/trash/getTrashCans/", inDB.MWCheckUserTokenCookie(), inDB.GetAllTrashCanByUser)

	// Authentication
	router.POST("/api/login", inDB.AuthLogin)
	router.POST("/api/register", inDB.AuthRegister)
	router.POST("/api/checkLogin", inDB.CheckAuth)
	router.POST("/api/logout", inDB.AuthLogout)
	router.GET("/", inDB.NotImplemented)

	// Admin
	router.Use(inDB.MWCheckUserTokenCookie())
	{
		router.GET("/api/getAllTrashVersion", inDB.GetAllTrashVersion)
		router.POST("/api/addTrashVersion", inDB.AddTrashVersion)
		router.PUT("/api/editTrashVersion/:trash_version_id", inDB.EditTrashVersion)
		router.DELETE("/api/deleteTrashVersion/:trash_version_id", inDB.DeleteTrashVersion)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	router.Run(":" + port)
}
