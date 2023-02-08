package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitRouter(router *gin.Engine) {
	//加载静态资源
	router.StaticFS("/js", http.Dir("../app"))
	router.LoadHTMLGlob("../app/*.html")

	//文件列表页面
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	/************************demo 页面实现的接口***********************************/
	//文件列表页面
	router.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	//文件展示页面
	router.GET("/view.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "view.html", gin.H{})
	})

	//前端获取dir文件夹下的文件名,仅供参考,开发者可以按照本身需求重新定义
	router.GET("/getListFile", GetListFileHandler)

	//文件下载,仅供参考,开发者可以按照本身需求重新定义
	router.GET("/demo/file/download", GetFileHanlder)

	//传入文件名,返回有效的url和token,仅供参考,开发者可以按照本身需求重新定义
	router.GET("/getUrlAndToken", GetUrlAndTokenHandler)

	//获取新建文件的模板页面,开发者可以按照本身需求重新定义
	router.GET("/getCreateFileUrlAndToken", GetCreateFileUrlAndTokenHandler)

	//生成url接口,仅提供参考,本demo未使用
	router.GET("/demo/url", GetUrlHandler)

	router.GET("/open", Open)
	/************************demo 页面实现的接口***********************************/

	/****************** 第三方根据实际业务情况实现的回调接口 *******************/
	r := router.Group("/v1/3rd")
	{
		r.Use(CheckOpenSignature)
		//r.Use(CheckToken)
		//r.Use(CheckFileid)
		//r.Use(CheckUserAgent)
		//r.Use(CheckAppid)

		//1.获取文件元数据
		r.GET("/file/info", FileHandler)

		//2.获取用户信息
		r.POST("/user/info", GetUserBatch)

		//3.上传文件新版本
		r.POST("/file/save", PostSaveFile)

		//4.上传在线编辑用户信息
		r.POST("/file/online", PostFileOnline)

		//5.获取特定版本的文件信息
		r.GET("/file/version/:version", FileVersion)

		//6.文件重命名
		r.PUT("/file/rename", PutFileName)

		//7.获取所有历史版本文件信息
		r.POST("/file/history", GetFileHistoryVersions)

		//9. 获取企业状态
		r.POST("/onnotify", PostNotification)

	}
	//8.新建文件
	router.POST("/v1/3rd/file/new", PostNewFile)

	/****************** 第三方根据实际业务情况实现的回调接口 *******************/
}
