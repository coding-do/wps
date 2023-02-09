# wpsweb-demo运行说明(当前版本1.0.9)

「WebOffice 知识库」文档： https://wwo.wps.cn/docs/front-end/high-frequency-usage/
「WPS开发平台」文档：https://editzt.myones.net/open#/docs/server/app-manage/api-list


**1.设置配置文件weboffice-demo.conf**

> port:     demo服务端口
>
> domain:   金山文档在线编辑域名(不需要修改)
>
> appid:    开发信息中的APPID
>
> appkey:   开发信息中的APPKEY
>
> download_host: 文件下载地址.即下载接口/v1/file的地址.(如:http://www.xxxx.com,需能被外网能访问到)
>
> local_dir: 文件路径位置

**2.终端运行demo命令:**

> cd wpsweb-demo/go_demo
>
> go build -o wpsweb-demo ./
>
> ./wpsweb-demo
>
> [提示: 若需要交叉编译,使用
> CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wpsweb-demo ./  ]

**3.打开浏览器输入地址:**

> demoIP:端口/index
>
> 如 http://www.xxxx.com:19999/index
