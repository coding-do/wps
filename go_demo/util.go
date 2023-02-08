package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

const (
	ERR_OK               int32 = 0     //服务处理正常
	ERR_UserNotLogin     int32 = 40001 // 用户未登录
	ERR_SessionExpired   int32 = 40002 // token过期
	ERR_PermissionDenied int32 = 40003 // 用户无权限访问
	ERR_NotExists        int32 = 40004 //资源不存在
	ERR_InvalidArgument  int32 = 40005
)

const (
	OK                    = "OK"
	Unavailable           = "Unavailable"
	Conflict              = "conflict"
	Unknown               = "Unknown"
	NotExists             = "fileNotExists" //资源不存在
	AlreadyExists         = "AlreadyExists"
	EmptyFile             = "EmptyFile" //文件为空而不能进行某项操作
	InvalidArgument       = "InvalidArgument"
	UserNotLogin          = "userNotLogin"
	UnknownApp            = "UnknownApp"
	FileTooLarge          = "fileTooLarge"
	PermissionDenied      = "permissionDenied"
	FileUploadNotComplete = "fileUploadNotComplete"
	FileUnderReview       = "FileUnderReview"      //审核中
	InternalError         = "InternalError"        //内部错误，不方便告诉你原因
	Canceled              = "Canceled"             //一般是用户断开网络连接，导致请求被取消
	FileDownloadCanceled  = "FileDownloadCanceled" //等待文件下载时，用户断开了连接
	FileOpenCanceled      = "FileOpenCanceled"     //等待文件打开时，用户断开了连接
	FileSizeLimit         = "fileSizeLimit"        // 大小限制
	Timeout               = "Timeout"
	SessionExpired        = "SessionExpired"
	NeedIdentityVerify    = "NeedIdentityVerify" //需要进行实名认证
	ClipboardExpire       = "ClipboardExpire"
	DownloadFailed        = "DownloadFailed"
	SignatureMismatch     = "InvalidSignature" //签名错误
	LightlinkForbid       = "lightlinkForbid"  //包含敏感信息
	NoChanges             = "NoChanges"
	UndoUnavailable       = "UndoUnavailable"
	SavedEmptyFile        = "SavedEmptyFile"
)

type JSONCodeError struct {
	Code    int32  `json:"code,omitempty"`
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

func ErrorMessage(c *gin.Context, statusCode int, code int32, result, msg string) {
	c.JSON(statusCode, &JSONCodeError{
		Code:    code,
		Result:  result,
		Message: msg,
	})
}

type Any interface{}

// JSONError golint
type JSONError struct {
	StatusCode int         `json:"-"`
	Result     string      `json:"result"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
}

// Error golint
func (err *JSONError) Error() string {
	return err.Result
}

// AbortWithErrorMessage golint
func AbortWithErrorMessage(c *gin.Context, status int, result string, message string) {
	err := &JSONError{
		StatusCode: status,
		Result:     result,
		Message:    message,
	}
	AbortWithError(c, err)
}

// AbortWithError golint
func AbortWithError(c *gin.Context, err error) {
	jsonErr, ok := err.(*JSONError)
	if !ok {
		jsonErr = &JSONError{
			StatusCode: http.StatusInternalServerError,
			Result:     "Unknown",
			Message:    err.Error(),
		}
	}
	c.JSON(jsonErr.StatusCode, jsonErr)
	c.Abort()
}

func pathExist(_path string) bool {
	indexMatches, err := filepath.Glob(_path)
	return err == nil && len(indexMatches) > 0
}

func Sign(values url.Values, secretKey string) string {
	contents := StringToSign(values, secretKey)
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write([]byte(contents))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	fmt.Println("signature:", signature)
	return signature
}

func StringToSign(values url.Values, secretKey string) []byte {
	keys := []string{}
	for k, _ := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := &bytes.Buffer{}
	for _, k := range keys {
		fmt.Fprintf(buf, "%s=%s", k, values.Get(k))
	}
	fmt.Fprintf(buf, "_w_secretkey=%s", secretKey)
	return buf.Bytes()
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func GetLatestVersion2(ctx context.Context, fname string) int32 {
	var max int
	files, _ := ioutil.ReadDir(App.LocalDir)
	for _, file := range files {
		if file.IsDir() {
			version, _ := strconv.Atoi(file.Name())
			max = Max(version, max)
		}
	}
	for i := 1; i <= max; i++ {
		fpath := filepath.Join(App.LocalDir, strconv.Itoa(i), fname)
		if !pathExist(fpath) {
			return int32(i)
		}
	}
	return int32(max)
}

func GetLatestVersion(fileid string) (int32, error) {
	fname, _ := filemap[fileid]
	if fname == "" {
		return 0, fmt.Errorf("invalid fileid:%s", fileid)
	}

	fileName, _, err := splitFileNameAndType(fname)
	if err != nil {
		return 0, err
	}

	path := filepath.Join(App.LocalDir, fileName)
	if !pathExist(path) {
		return 0, fmt.Errorf("file is not exist , path:%s", path)
	}

	var max int
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {
		filename, _, err := splitFileNameAndType(file.Name())
		if err != nil {
			continue
		}
		version, err := strconv.Atoi(filename)
		if err != nil {
			continue
		}

		max = Max(version, max)
	}
	if max == 0 {
		return 0, fmt.Errorf("file exist, fileid:%s", fileid)
	}
	return int32(max), nil
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func NoHyphenString(u uuid.UUID) string {
	return fmt.Sprintf("%x%x%x%x%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func GetContextFileid(c *gin.Context) string {
	return c.Request.Header.Get(XFileID)
}

func SetContextFileid(c *gin.Context, fileid string) {
	c.Set(XFileID, fileid)
}

func getFileDirPath(fileid string) (string, error) {
	fname, _ := filemap[fileid]
	if fname == "" {
		return "", fmt.Errorf("fileid is not exist")
	}

	arr := strings.Split(fname, ".")
	if len(arr) < 2 {
		return "", fmt.Errorf("invalid fname ")
	}
	return filepath.Join(App.LocalDir, strings.Join(arr[:len(arr)-1], ".")), nil
}

//fname = 文字测试文件.docx
//return "文字测试文件","docx",err
func splitFileNameAndType(fname string) (string, string, error) {
	arr := strings.Split(fname, ".")
	if len(arr) < 2 {
		return "", "", fmt.Errorf("invalid fname ")
	}

	return strings.Join(arr[:len(arr)-1], "."), arr[len(arr)-1], nil
}

func getFileDirPathOrMkdir(fileid string) (string, error) {
	fname, _ := filemap[fileid]
	if fname == "" {
		return "", fmt.Errorf("fileid is not exist")
	}

	fileName, fileType, err := splitFileNameAndType(fname)
	if err != nil {
		return "", err
	}

	path := filepath.Join(App.LocalDir, fileName)
	if !pathExist(path) {
		err := os.Mkdir(path, 0777)
		if err != nil {
			fmt.Println("create file fail: ", err.Error())
			return "", err
		}
		historypath := filepath.Join(path, fmt.Sprintf("%d.%s", 1, fileType))
		fpath := filepath.Join(App.LocalDir, fname)
		_, err = CopyFile(historypath, fpath)
		return "", err
	}
	return path, nil
}

func GetFileSize(path string) (int64, error) {
	file, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return file.Size(), nil
}

func mkdirDocument(path string) error {
	if !pathExist(path) {
		err := os.Mkdir(path, 0777)
		if err != nil {
			fmt.Println("create file fail: ", err.Error())
			return err
		}
	}
	return nil
}

func GetFileidHash(name string) string {
	number := GetHash([]byte(name))
	return fmt.Sprintf("%d", number)
}

const (
	c1_32 uint32 = 0xcc9e2d51
	c2_32 uint32 = 0x1b873593
)

// GetHash returns a murmur32 hash for the data slice.
func GetHash(data []byte) uint32 {
	// Seed is set to 37, same as C# version of emitter
	var h1 uint32 = 37

	nblocks := len(data) / 4
	var p uintptr
	if len(data) > 0 {
		p = uintptr(unsafe.Pointer(&data[0]))
	}

	p1 := p + uintptr(4*nblocks)
	for ; p < p1; p += 4 {
		k1 := *(*uint32)(unsafe.Pointer(p))

		k1 *= c1_32
		k1 = (k1 << 15) | (k1 >> 17) // rotl32(k1, 15)
		k1 *= c2_32

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) // rotl32(h1, 13)
		h1 = h1*5 + 0xe6546b64
	}

	tail := data[nblocks*4:]

	var k1 uint32
	switch len(tail) & 3 {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1_32
		k1 = (k1 << 15) | (k1 >> 17) // rotl32(k1, 15)
		k1 *= c2_32
		h1 ^= k1
	}

	h1 ^= uint32(len(data))

	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16

	return (h1 << 24) | (((h1 >> 8) << 16) & 0xFF0000) | (((h1 >> 16) << 8) & 0xFF00) | (h1 >> 24)
}

func RandString(n int) string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ss := make([]byte, n)
	for i := 0; i < n; i++ {
		b := r.Intn(26) + 65
		ss[i] = byte(b)
	}
	return string(ss)
}
