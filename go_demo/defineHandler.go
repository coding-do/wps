package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/bangwork/wiki-api/app/utils/errors"
	"github.com/kun/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//token存活时间
const TokenExpiresTime = 600

func GetListFileHandler(c *gin.Context) {
	getListFile(App.LocalDir)
	fmt.Println("listFile:", filemap)
	c.JSON(http.StatusOK, gin.H{
		"name_files": filemap,
	})
	return
}

//返回路径下,以(docx|pptx|xlsx)结尾的文件名
func getListFile(path string) (map[string]string, error) {
	filemap = make(map[string]string)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("read dir fail:", err)
		return nil, err
	}

	for _, file := range files {
		//过滤文件
		var ref = regexp.MustCompile(".(docx|pptx|xlsx|pdf)$")
		if ref.MatchString(file.Name()) {
			//demo用于测试,注意fileid生成需对接企业自己生成与管理
			//正式环境中,保证一个文件一个fileid,且fileid不会改变
			fmt.Println("file.Name() ====", file.Name())
			file_id := GetFileidHash(file.Name())
			filemap[file_id] = file.Name()
			fmt.Println("fileid:", file_id, "    file_name:", file.Name())
		}
	}
	return filemap, nil
}

func GetUrlAndTokenHandler(c *gin.Context) {
	fileid := c.Query("fileid")
	if fileid == "" {
		fmt.Println("error : missing parameter fileid")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter fileid")
		return
	}

	wpsUrl, err := getWpsUrl(fileid)
	if err != nil {
		fmt.Println("err :", err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	//第一次访问,新建文件夹
	_, err = getFileDirPathOrMkdir(fileid)
	if err != nil {
		fmt.Println("err :", err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token.GetTokenKey(),
		"expires_in": TokenExpiresTime,
		"wpsUrl":     wpsUrl,
	})
	return
}

func GetCreateFileUrlAndTokenHandler(c *gin.Context) {

	_w_filetype := c.Query("_w_filetype")
	if _w_filetype == "" {
		fmt.Println("error : missing parameter _w_filetype")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter _w_filetype")
		return
	}

	var values = make(url.Values)
	values.Add("_w_appid", App.Appid)
	values.Add("_w_tokentype", "1") //必须携带,回调才会有token
	values.Add("_w_filetype", _w_filetype)
	signature := Sign(values, App.Appkey)

	newFileURL := fmt.Sprintf("%s/weboffice/office/%s/3587629245?%s&_w_signature=%s", App.Domain, _w_filetype, values.Encode(), url.QueryEscape(signature))
	fmt.Println("newFileURL:", newFileURL)

	//https://wwo.wps.cn/office/w/new/0?_w_appid=56a92fa1b489492c78afc69a39fb2a61&_w_signature=%2BEmG5aChsWK3%2F4gvIpjjf2n8sbs%3D&_w_tokentype=1

	c.JSON(http.StatusOK, gin.H{
		"token":      token.GetTokenKey(),
		"expires_in": TokenExpiresTime,
		"wpsUrl":     newFileURL,
	})
	return
}

func getWpsUrl(fileid string) (string, error) {
	fname, _ := filemap[fileid]
	if fname == "" {
		return "", fmt.Errorf("fileid is not exist")
	}

	//t:文件类型
	arr := strings.Split(fname, ".")
	t := editorExtMap[arr[len(arr)-1]]
	//https://editzt.ones.ai/weboffice/office/s/19e64d48d9c7a39d1fda5affef14756a?
	//_w_appid=AK20210702WEFYET
	//&_w_third_appid=AK20220613GULYER
	//&_w_third_file_id=RqP3GJK7
	//&_w_tokentype=1
	//&_w_token=user_uuid=QYTt3ymn,team_uuid=RDjYMhKq,token=iQhLmcG9Oudav0QpwBK3qImEk3bPlBSBovzpMEbAm0JSRbIvS2PoRADyeFi3yvVF&lang=zh-CN
	//默认参数,需根据需求修改
	var values = make(url.Values)
	values.Add("_w_appid", App.Appid)
	values.Add("_w_appid", App.Appid)

	values.Add("_w_tokentype", "1") //必须携带,回调才会有token
	//values.Add("param", "xxxx")

	signature := Sign(values, App.Appkey)
	webofficeUrl := fmt.Sprintf("%s/weboffice/office/%s/%s?%s&_w_signature=%s", App.Domain, t, fileid, values.Encode(), url.QueryEscape(signature))
	return webofficeUrl, nil
}

//下载文件
func GetFileHanlder(c *gin.Context) {
	fileid := c.Query("_w_fileid")
	if fileid == "" {
		fmt.Println("error : missing param fileid")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter fileid")
		return
	}

	fname := filemap[fileid]
	if fname == "" {
		fmt.Println("err : fileid not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", fileid))
		return
	}

	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Println("error:", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	version, _ := strconv.Atoi(c.Query("version"))
	if version == 0 {
		ver, err := GetLatestVersion(fileid)
		if err != nil {
			fmt.Println("error:", err)
			ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
			return
		}

		version = int(ver)
	}
	verName := fmt.Sprintf("%d.%s", version, fileType)
	fpath := filepath.Join(App.LocalDir, fileName, verName)
	fmt.Println("fpath:", fpath)
	if !pathExist(fpath) {
		fmt.Printf("file not exist: %v \n", fpath)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("%s:filepathNotExists", fpath))
		return
	}
	c.Writer.Header().Add("Content-Disposition", "attachment;filename="+verName)
	c.Writer.Header().Add("Content-Type", "application/octet-stream")
	c.File(fpath)
}

type OpSTR struct {
	Result int    `json:"result"`
	Url    string `json:"url"`
}

func Open(c *gin.Context) {

	fileid := c.Query("fileid")
	if fileid == "" {
		fmt.Println("error : missing param fileid")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter fileid")
		return
	}

	getToken, _ := RequestGetToken()
	client := http.Client{Timeout: 3 * time.Second}
	body := make(map[string]interface{})
	body["app_token"] = getToken.AppToken
	body["file_id"] = fileid
	body["type"] = "s"
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/open/weboffice/v1/url", App.Domain), bytes.NewReader(raw))
	if err := WPS3Sign(req, raw, App.Appid, App.Appkey); err != nil {
		fmt.Println("WPS3Sign ==== >")
	}
	resp, err := client.Do(req)
	fmt.Println("err===", err)
	respBody, err1 := ioutil.ReadAll(resp.Body)

	fmt.Println("err1===", err1)
	fmt.Println(string(respBody))

	wt := new(OpSTR)
	err = json.Unmarshal(respBody, wt)

	c.JSON(http.StatusOK, gin.H{
		"token":      token.GetTokenKey(),
		"expires_in": TokenExpiresTime,
		"wpsUrl":     wt.Url,
	})
	return

}

func WPS3Sign(request *http.Request, data []byte, appId, appKey string) error {
	contentType := "application/json"
	m := md5.New()
	m.Write(data)
	contentMd5 := fmt.Sprintf("%x", m.Sum(nil))
	date := time.Now().UTC().Format(http.TimeFormat)
	s := sha1.New()
	io.WriteString(s, strings.ToLower(appKey))
	io.WriteString(s, contentMd5)
	// 签名 /open/ 之后的 path
	urlParts := strings.Split(request.URL.RequestURI(), "/open/")
	if len(urlParts) != 2 {
		return errors.Trace(fmt.Errorf("WPS3Sign url error, url should have two parts, split by /open/, url=%s", request.URL.RequestURI()))
	}
	io.WriteString(s, "/"+urlParts[1])
	io.WriteString(s, contentType)
	io.WriteString(s, date)

	sign := fmt.Sprintf("%x", s.Sum(nil))
	sign = fmt.Sprintf("WPS-3:%s:%s", appId, sign)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Content-Md5", contentMd5)
	request.Header.Set("Date", date)
	request.Header.Set("X-Auth", sign)
	return nil
}

type WpsMidToken struct {
	Result int     `json:"result"`
	Token  *Token2 `json:"token"`
}

type Token2 struct {
	AppToken  string `json:"app_token"`
	ExpiresIn int64  `json:"expires_in"`
}

func RequestGetToken() (*Token2, error) {
	client := http.Client{Timeout: 3 * time.Second}
	body := make(map[string]interface{})
	body["app_id"] = App.Appid
	body["scope"] = "file_edit"
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/open/auth/v1/app/inscope/token", App.Domain), bytes.NewReader(raw))
	if err := WPS3Sign(req, raw, App.Appid, App.Appkey); err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	wt := new(WpsMidToken)
	err = json.Unmarshal(respBody, wt)
	if err != nil {
		log.Info("WpsMidToken", string(respBody))
		return nil, errors.Trace(err)
	}
	if wt.Token == nil {
		log.Info("WpsMidToken", string(respBody))
		return nil, errors.P(errors.Wps, errors.Config, errors.InvalidFormat)
	}

	fmt.Println(" wt.Token ==", wt.Token)
	return wt.Token, nil
}

func GetUrlHandler(c *gin.Context) {
	file_id := c.Request.Header.Get(XFileID)
	if file_id == "" {
		fmt.Println("error : fileid is not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter fileid")
		return
	}

	fname := filemap[file_id]
	if fname == "" {
		fmt.Println("err : fileid not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", file_id))
		return
	}

	_, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Println("error : ", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	t := editorExtMap[fileType]
	query := c.Request.URL.Query()
	query.Add("_w_tokentype", "1")
	//剔除非wps的参数
	for k, _ := range query {
		if !re.MatchString(k) {
			query.Del(k)
		}
	}
	signature := Sign(query, App.Appkey)
	webofficeUrl := fmt.Sprintf("%s/weboffice/office/%s/%s?%s&_w_signature=%s", App.Domain, t, file_id, c.Request.URL.RawQuery, url.QueryEscape(signature))
	c.String(http.StatusOK, webofficeUrl)
}
