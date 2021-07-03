/**
  @author:panliang
  @data:2021/6/18
  @note
**/
package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	Auth "go_im/bin/http/controller/auth"
	"go_im/bin/http/middleware"
)

var router *gin.Engine

func RegisterApiRoutes(router *gin.Engine) {
	//允许跨域
	weibo := new(Auth.WeiBoController)
	auth := new(Auth.AuthController)
	users := new(Auth.UsersController)

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                                                                 //允许所有域名
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}                                                                      //允许请求的方法
	config.AllowHeaders = []string{"tus-resumable", "upload-length", "upload-metadata", "cache-control", "x-requested-with", "*"} //允许的Header
	router.Use(cors.New(config))

	//router.Use(middleware.CrosHandler())
	router.GET("/", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"message": "hello world!!!",
		})
	})

	router.GET("/api/WeiBoCallBack", weibo.WeiBoCallBack)
	router.GET("/api/giteeCallBack", auth.GiteeCallBack)
	api := router.Group("/api").Use(middleware.Auth())
	{
		api.POST("/me", auth.Me)
		api.POST("/refresh", auth.Refresh)
		api.GET("/UsersList", users.GetUsersList)
		api.GET("/InformationHistory", users.InformationHistory)
	}
}
