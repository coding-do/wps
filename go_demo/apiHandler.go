package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

//虚拟参数
const (
	creator    = "999"
	createTime = 1136185445
)

var re = regexp.MustCompile("^_w_")

//支持的文件格式
var etExts = []string{"et", "xls", "xlt", "xlsx", "xlsm", "xltx", "xltm", "csv"}
var wpsExts = []string{"doc", "docx", "txt", "dot", "wps", "wpt", "dotx", "docm", "dotm"}
var wppExts = []string{"ppt", "pptx", "pptm", "pptm", "ppsm", "pps", "potx", "potm", "dpt", "dps"}
var pdfExts = []string{"pdf"}
var editorExtMap = map[string]string{}

func init() {
	for _, ext := range etExts {
		editorExtMap[ext] = "s"
	}
	for _, ext := range wpsExts {
		editorExtMap[ext] = "w"
	}
	for _, ext := range wppExts {
		editorExtMap[ext] = "p"
	}
	for _, ext := range pdfExts {
		editorExtMap[ext] = "f"
	}
}

//1.获取文件元数据
func FileHandler(c *gin.Context) {
	file_id := GetContextFileid(c)
	fname := filemap[file_id]
	if fname == "" {
		fmt.Println("err : file not exist", file_id)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", file_id))
		return
	}

	fmt.Println("######1111")
	fpath := filepath.Join(App.LocalDir, fname)
	if !pathExist(fpath) {
		fmt.Printf("file not exist: %v \n", fpath)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("%s:filepathNotExists", fpath))
		return
	}

	fmt.Println("#####22222")

	rand.Seed(time.Now().UnixNano())
	uid := fmt.Sprintf("%d", rand.Intn(100))

	//获取最新版本号
	version, err := GetLatestVersion(file_id)
	if err != nil {
		fmt.Printf("FileHandler error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	fmt.Println("#####3333")
	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Printf("FileHandler error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	newVerName := fmt.Sprintf("%d.%s", version, fileType)
	newFilePath := filepath.Join(App.LocalDir, fileName, newVerName)

	fmt.Println("newFilePath=====》", newFilePath)

	fileSize, err := GetFileSize(newFilePath)
	if err != nil {
		fmt.Printf("FileHandler error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	userAcl := &UserACL{
		Rename:  1,
		History: 1,
	}

	file := &FileModel{
		Id:          file_id,
		Name:        fname,
		Size:        fileSize,
		DownloadUrl: fmt.Sprintf("%s/demo/file/download?_w_fileid=%s", App.DownloadHost, file_id),
		Creator:     creator,
		CreateTime:  createTime,
		Modifier:    uid,
		ModifyTime:  time.Now().Unix(),
		Version:     version,
		UserAcl:     userAcl,
		Watermark: &Watermark{
			Type:  1,            //水印类型， 0为无水印； 1为文字水印
			Value: "这是一个测试环境XS", //文字水印的文字，当type为1时此字段必选
		},
	}
	user := &UserModel{
		Id:         uid,
		Permission: "write", // or "read",
		AvatarUrl:  "https://picsum.photos/100/100/?image=" + uid,
		Name:       "wps_user-" + uid,
	}
	fem := &FileEditModel{
		File: *file,
		User: *user,
	}
	c.JSON(http.StatusOK, fem)
}

//2.获取用户信息
func GetUserBatch(c *gin.Context) {
	var users []*UserModel
	var in GetUserInfoBatchInput
	err := c.BindJSON(&in)
	if err != nil {
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	for _, id := range in.Ids {
		var User UserModel
		User.Id = id
		User.Permission = "write"
		uid, _ := strconv.Atoi(id)
		User.AvatarUrl = fmt.Sprintf("https://picsum.photos/100/100/?image=%d", uid)
		User.Name = fmt.Sprintf("wps_user-%d", uid)
		users = append(users, &User)
	}
	out := GetUserInfoBatchOutput{
		Users: users,
	}
	c.JSON(http.StatusOK, out)
}

//3.上传文件新版本
//发者根据自己需求去修改功能控制
func PostSaveFile(c *gin.Context) {
	fileid := GetContextFileid(c)
	fname := filemap[fileid]
	if fname == "" {
		fmt.Println("err : file not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", fileid))
		return
	}
	uid := "100"

	path, err := getFileDirPathOrMkdir(fileid)
	if err != nil {
		fmt.Printf("path:%s   err:%s \n", path, err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	LatestVersion, err := GetLatestVersion(fileid)
	if err != nil {
		fmt.Printf("error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}
	newVersion := LatestVersion + 1

	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Printf("PostSaveFile error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	newVerName := fmt.Sprintf("%d.%s", newVersion, fileType)
	newFilePath := filepath.Join(App.LocalDir, fileName, newVerName)
	fmt.Println(newFilePath)
	newFile, err := os.Create(newFilePath)
	if err != nil {
		fmt.Println("create file error:", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}
	defer newFile.Close()

	//更新文档
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Println("get form file faild: ", err.Error())
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "invalid param file")
		return
	}

	// Copy数据
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("copy file faild: ", err.Error())
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	fileSize, err := GetFileSize(newFilePath)
	if err != nil {
		fmt.Printf("PostSaveFile error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	fm := &FileModel{
		Id:          fileid,
		Name:        fname,
		Size:        fileSize,
		DownloadUrl: App.DownloadHost + "/demo/file/download?_w_fileid=" + fileid,
		Creator:     creator,
		CreateTime:  createTime,
		Modifier:    uid,
		ModifyTime:  time.Now().Unix(),
		Version:     newVersion,
	}
	c.JSON(http.StatusOK, gin.H{
		"file": fm,
	})
}

//4.上传在线编辑用户信息
func PostFileOnline(c *gin.Context) {
	var in GetUserInfoBatchInput
	var users []*UserModel
	err := c.BindJSON(&in)
	if err != nil {
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}
	for _, id := range in.Ids {
		user := &UserModel{
			Id:         id,
			Permission: "write",
			AvatarUrl:  "https://picsum.photos/100/100/?image=" + id,
			Name:       "wps_user-" + id,
		}
		users = append(users, user)
	}
	c.String(http.StatusOK, "success")
}

//5.获取特定版本的文件信息
func FileVersion(c *gin.Context) {
	fileid := GetContextFileid(c)

	fname := filemap[fileid]
	if fname == "" {
		fmt.Println("err : file not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", fileid))
		return
	}
	uid := "100"
	version, _ := strconv.Atoi(c.Param("version"))

	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Printf("FileVersion error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	newVerName := fmt.Sprintf("%d.%s", version, fileType)
	newFilePath := filepath.Join(App.LocalDir, fileName, newVerName)

	fileSize, err := GetFileSize(newFilePath)
	if err != nil {
		fmt.Printf("FileVersion error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}
	file := &FileModel{
		Id:          fileid,
		Name:        fname,
		Size:        fileSize,
		DownloadUrl: fmt.Sprintf("%s/demo/file/download?_w_fileid=%s&version=%d", App.DownloadHost, fileid, version),
		Creator:     creator,
		CreateTime:  createTime,
		Modifier:    uid,
		ModifyTime:  time.Now().Unix(),
		Version:     int32(version),
	}
	out := &GetFileVersionOutput{
		File: file,
	}
	c.JSON(http.StatusOK, out)
}

//6.文件重命名
func PutFileName(c *gin.Context) {
	var args PutFileInput
	c.BindJSON(&args)
	if args.Name == "" {
		fmt.Println("error: invalid param name")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter name")
		return
	}

	fileid := GetContextFileid(c)
	fname := filemap[fileid]
	if fname == "" {
		fmt.Println("err : file not exist")
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, fmt.Sprintf("fileid:%s not exist", fileid))
		return
	}

	oldfileName, _, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Println("error :", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	newfileName, _, err := splitFileNameAndType(args.Name)
	if err != nil {
		fmt.Println("error :", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	newFilePath := filepath.Join(App.LocalDir, args.Name)
	newFileVersionPath := filepath.Join(App.LocalDir, newfileName)

	oldFilePath := filepath.Join(App.LocalDir, fname)
	oldFileVersionPath := filepath.Join(App.LocalDir, oldfileName)

	//修改原文件名称
	if err = os.Rename(oldFilePath, newFilePath); err != nil {
		fmt.Printf("rename  oldpath:%s   newpaht:%s err : %v\n", oldFilePath, newFilePath, err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	//修改版本文件夹名称
	if err := os.Rename(oldFileVersionPath, newFileVersionPath); err != nil {
		fmt.Printf("rename  oldpath:%s   newpaht:%s err : %v\n", oldFileVersionPath, newFileVersionPath, err)
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, err.Error())
		return
	}

	filemap[fileid] = args.Name
	c.Status(http.StatusOK)
}

//7.获取所有历史版本文件信息
func GetFileHistoryVersions(c *gin.Context) {
	var in GetFileHistoryVersionsRequest
	err := c.BindJSON(&in)
	if err != nil {
		fmt.Println("error:", err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	fileid := GetContextFileid(c)

	fname := filemap[fileid]
	if fname == "" {
		fmt.Println("err : invalid fileid")
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "invalid fileid")
		return
	}

	uid := "100"
	version, err := GetLatestVersion(fileid)
	if err != nil {
		fmt.Printf("GetFileHistoryVersions error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}

	history := []*FileMetadata{}
	count := in.Count
	if count > version {
		count = version
	}

	if in.Offset == 0 {
		in.Offset = 1
	}
	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		fmt.Printf("GetFileHistoryVersions error: %v \n", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}
	for i := count; i >= in.Offset; i-- {
		fmt.Println("history version :", i)

		newVerName := fmt.Sprintf("%d.%s", i, fileType)
		newFilePath := filepath.Join(App.LocalDir, fileName, newVerName)

		fileSize, err := GetFileSize(newFilePath)
		if err != nil {
			fmt.Printf("GetFileHistoryVersions error: %v \n", err)
			ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
			return
		}
		md := &FileMetadata{
			Id:          fileid,
			Name:        fname,
			Size:        fileSize,
			DownloadUrl: fmt.Sprintf("%s/demo/file/download?_w_fileid=%s&version=%d", App.DownloadHost, fileid, int32(i)),
			Version:     int32(i),
			Type:        "file",
			CreateTime:  createTime,
			ModifyTime:  time.Now().Unix(),
		}
		md.Creator = &UserModel{
			Id:        creator,
			Name:      "wps_user-" + creator,
			AvatarUrl: "https://picsum.photos/100/100/?image=" + creator,
		}
		md.Modifier = &UserModel{
			Id:        uid,
			Name:      "wps_user-" + uid,
			AvatarUrl: "https://picsum.photos/100/100/?image=" + uid,
		}
		history = append(history, md)
	}

	out := GetFileHistoryVersionsResponse{
		Histories: history,
	}
	c.JSON(http.StatusOK, out)
}

//8.新建文件
func PostNewFile(c *gin.Context) {

	fmt.Println("8.新建文件----")

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Println("get form file faild: ", err.Error())
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter file")
		return
	}
	//文件名
	fname := c.Request.Form.Get("name")
	if fname == "" {
		ErrorMessage(c, http.StatusBadRequest, ERR_InvalidArgument, InvalidArgument, "missing parameter name")
		return
	}
	_, fileType, _ := splitFileNameAndType(fname)
	fname = RandString(10) + "." + fileType

	//新文件完整路径
	fpath := filepath.Join(App.LocalDir, fname)

	// 创建新文件
	newFile, err := os.Create(fpath)
	if err != nil {
		fmt.Println("create file error:", err)
		ErrorMessage(c, http.StatusBadRequest, ERR_NotExists, NotExists, err.Error())
		return
	}
	defer newFile.Close()
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("copy file faild: ", err.Error())
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	file_id := GetFileidHash(fname)
	filemap[file_id] = fname

	//新建文件夹
	_, err = getFileDirPathOrMkdir(file_id)
	if err != nil {
		fmt.Println("err :", err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}

	uid := "100"
	appid := c.Query("_w_appid")

	t := editorExtMap[fileType]

	query := fmt.Sprintf("_w_userid=%s&_w_appid=%s&_w_permission=write&_w_tokentype=1", uid, appid)
	urlquery, _ := url.ParseQuery(query)
	signature := Sign(urlquery, App.Appkey)
	redirectUrl := fmt.Sprintf("%s/weboffice/office/%s/%s?%s&_w_signature=%s", App.Domain, t, file_id, query, url.QueryEscape(signature))
	getTemplateInfo := GetTemplateInfo{
		RedirectUrl: redirectUrl,
		UserId:      uid,
	}

	//	utl := fmt.Sprintf("%s/weboffice/office/%s/%s?%s&_w_signature=%s", App.Domain, t, file_id, query, url.QueryEscape(signature))

	c.JSON(http.StatusOK, getTemplateInfo)
}

type PostNotificationReq struct {
	Cmd  string      `json:"cmd"`
	Body interface{} `json:"body"`
}

//9.获取企业信息
func PostNotification(c *gin.Context) {
	var in PostNotificationReq
	err := c.BindJSON(&in)
	if err != nil {
		fmt.Println("error:", err)
		ErrorMessage(c, http.StatusInternalServerError, ERR_InvalidArgument, InternalError, err.Error())
		return
	}
	fmt.Println("in:", in)

	c.JSON(http.StatusOK, gin.H{
		"msg": "success",
	})
	return
}
