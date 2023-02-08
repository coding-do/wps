package main

import (
	//"fmt"
	//"net/http"
	//"net/url"

	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	XFileID    = "x-weboffice-file-id"
	XUserToken = "x-wps-weboffice-token"
)

//检查token
func CheckToken(c *gin.Context) {
	token.RLock()
	defer token.RUnlock()
	fmt.Println("demo current token: ", token.key)

	access_token := c.Request.Header.Get(XUserToken)
	if access_token != token.key {
		fmt.Println("invalid access_token: ", access_token)
		AbortWithErrorMessage(c, http.StatusBadRequest, "InvalidToken", "invalid token")
		return
	}

	if token.timeout-time.Now().Unix() < 0 {
		fmt.Println(" error: Token Time Out")
		AbortWithErrorMessage(c, http.StatusBadRequest, "TokenTimeOut", "token is timeout")
		return
	}
}

//检查签名
func CheckOpenSignature(c *gin.Context) {
	query := c.Request.URL.Query()

	var re = regexp.MustCompile("^_w_")
	signQuery := url.Values{}
	for k, _ := range query {
		if re.MatchString(k) {
			signQuery.Set(k, query.Get(k))
		}
	}

	signature, err := url.PathUnescape(query.Get("_w_signature"))
	if err != nil {
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidRequest", "invaid signature")
		return
	}

	signQuery.Del("_w_signature")
	authorization := Sign(signQuery, App.Appkey)
	if authorization != signature {
		fmt.Printf("error : authorization:%s and signature:%s \n", authorization, signature)
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidSignature", "signature mismatch")
		return
	}
}

func CheckFileid(c *gin.Context) {
	fileid := c.Request.Header.Get(XFileID)
	fmt.Println("CheckFileid fileid:", fileid)
	if fileid == "" {
		fmt.Println("error : param filedi is nil")
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidParams", "header missing param x-weboffice-file-id")
		return
	}
	if !existFileid(fileid) {
		fmt.Println("error : fileid is not exist")
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidParams", "fileid is not exist")
		return
	}
	SetContextFileid(c, fileid)
	c.Next()
}

func CheckUserAgent(c *gin.Context) {
	agent := c.Request.Header.Get("User-Agent")
	fmt.Println("CheckUserAgent agent:", agent)
	if agent == "" {
		fmt.Println("error : agent is not exist")
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidParams", "header missing param User-Agent")
		return
	}
}

func CheckAppid(c *gin.Context) {
	query := c.Request.URL.Query()
	appid := query.Get("_w_appid")
	if appid == "" {
		AbortWithErrorMessage(c, http.StatusUnauthorized, "InvalidRequest", "missing param _w_appid")
		return
	}
}

func existFileid(fileid string) bool {
	_, ok := filemap[fileid]
	return ok
}
