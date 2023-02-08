package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/naoina/toml"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type AppConfig struct {
	DownloadHost string
	LocalDir     string
	Port         string
	Domain       string
	Appid        string
	Appkey       string
}
type Token struct {
	key     string
	timeout int64
	sync.RWMutex
}

var App AppConfig

var token *Token

func (token *Token) GetTokenKey() string {
	token.Lock()
	defer token.Unlock()
	//token超时或者未初始化,则生成一个新的token
	if token.key == "" || token.timeout-time.Now().Unix() <= 0 {
		uuid, _ := uuid.NewV4()
		token.key = NoHyphenString(uuid)
	}
	token.timeout = time.Now().Add(TokenExpiresTime * time.Second).Unix()
	fmt.Println("GetTokenKey:", token.key)
	return token.key
}

//对接模块,需自己修改,持久化管理fild
var filemap = make(map[string]string)

func initConfig(file string) error {
	// 读取配置文件
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if err := toml.Unmarshal(buf, &App); err != nil {
		return err
	}
	return nil
}

func main() {

	token = &Token{}

	err := initConfig("../weboffice-demo.conf")
	if err != nil {
		fmt.Println("init config faild: %v", err.Error())
		os.Exit(2)
	}
	rDefault := gin.Default()
	InitRouter(rDefault)
	rDefault.Run(":" + App.Port)
}
