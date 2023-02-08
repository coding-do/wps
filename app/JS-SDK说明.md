# WPS Web Office JS-SDK 说明文档

## 概述

WPS Web Office JS-SDK 是[WPS开放平台](https://open.wps.cn/) 面向网页开发者提供的网页开发工具包。

通过使用 JS-SDK，网页开发者可以为 WPS Web Office 自定义菜单、分享等能力，同时可以直接使用高级 API 来操作文档，为用户提供更优质的网页体验。

此文档面向网页开发者介绍 WPS Web Office JS-SDK 如何使用及相关注意事项。

## JSSDK 使用

### 引入 JS 文件 (当前版本1.0.9)

在需要调用的页面引入如下 JS 文件(jwps.js)

备注：支持使用 AMD/CMD 标准模块加载方法加载

### 接入 WPS Web Office

```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?_w_appid=xxx&_w_id=xxx&_w_type=1&_w_userid=xxx&_w_timestamp=xxx&_w_permission=write&_w_tokentype=1&&_w_signature=xxx' // 如文字(Word)接入地址
})
```
JS-SDK 会自动创建 iframe(#wps-iframe)，它默认会挂载到 body 下。

### 设置 token

> url参数带上_w_tokentype=1（此参数同样需要签名）
```js
wps = WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?_w_appid=xxx&_w_id=xxx&_w_type=1&_w_userid=xxx&_w_timestamp=xxx&_w_permission=write&_w_tokentype=1&_w_signature=xxx'
})

// 获取到 token 后，设置
wps.setToken({token: 'your-token'})
```

### 自定义 Web Office(iframe) 挂载点

```js
WPS.config({
  mount: document.querySelector('#container'),
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx'
})
```

### 头部配置

#### 分享按钮

```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx',
  headers: {
    shareBtn: {
      tooltip: '分享',
      subscribe(wps) {
        console.log(wps)
      }
    }
  }
})
```

#### 左边其他按钮（配置自定义菜单）

```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx',
  headers: {
    otherMenuBtn: {
      tooltip: '更多菜单',
      items: [
        {
          type: 'custom',
          icon: 'http://ep.wps.cn/index/images/logo_white2.png', // 移动端显示 Icon
          text: '自定义菜单',
          subscribe(wps) {
            console.log(wps)
          }
        }
      ]
    }
  }
})
```

### 高级 API

#### 导出 PDF

```js
// 文字
wps.WpsApplication().ActiveDocument.ExportAsFixedFormatAsync() // 导出所有

// 表格
wps.EtApplication().ActiveWorkbook.ExportAsFixedFormatAsync() // 导出所有工作表
wps.EtApplication().ActiveWorkbook.ActiveSheet.ExportAsFixedFormatAsync() // 导出当前工作表

// 演示
wps.WppApplication().ActivePresentation.ExportAsFixedFormatAsync() // 导出所有
```

示例：自定义菜单中使用高级 API

```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx',
  headers: {
    otherMenuBtn: {
      tooltip: '更多菜单',
      items: [
        {
          type: 'custom',
          icon: 'http://ep.wps.cn/index/images/logo_white2.png',// 移动端显示的 Icon
          text: 'API 导出 PDF',
          async subscribe(wps) {
            let result = ""
            if (wps.WpsApplication) { // 文字
              result = await wps.WpsApplication().ActiveDocument.ExportAsFixedFormatAsync()
              console.table(result)
            }
            if (wps.EtApplication) { // 表格
              result = await wps.EtApplication().ActiveWorkbook.ExportAsFixedFormatAsync()
              console.table(result)
              result = await wps.EtApplication().ActiveWorkbook.ActiveSheet.ExportAsFixedFormatAsync()
              console.table(result)
            }
            if (wps.WppApplication) { // 演示
              result = await wps.WppApplication().ActivePresentation.ExportAsFixedFormatAsync()
              console.table(result)
            }
          }
        }
      ]
    }
  }
})
```

示例：配置完直接测试高级 API
```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx',
})

await wps.ready() // 需要等待 web office 内核初始化完成

let result = ""
if (wps.WpsApplication) { // 文字
  result = await wps.WpsApplication().ActiveDocument.ExportAsFixedFormatAsync()
  console.table(result)
}
if (wps.EtApplication) { // 表格
  result = await wps.EtApplication().ActiveWorkbook.ExportAsFixedFormatAsync()
  console.table(result)
  result = await wps.EtApplication().ActiveWorkbook.ActiveSheet.ExportAsFixedFormatAsync()
  console.table(result)
}
if (wps.WppApplication) { // 演示
  result = await wps.WppApplication().ActivePresentation.ExportAsFixedFormatAsync()
  console.table(result)
}
```


#### 保存版本

```js
WPS.config({
  wpsUrl: 'https://wwo.wps.cn/office/w/66858743377433009?xxx',
})

await wps.ready() // 需要等待 web office 内核初始化完成

await wps.save() // 保存版本
```

### 事件监听

通过wps.on 方法进行事件监听
```javascript
wps.on("事件名", function(data){
    // ....
})

```

#### 打开文件
```javascript
wps.on("fileOpen", function(data){
    // ....
})

```
文档呈现结果，无论成功或者失败，都会触发该事件,回调数据格式如下

```javascript
// 打开并渲染成功
{
    success: true,
    time: 111
}
// 打开失败
{
    {
        success: false, 
        time: 2000, // 当前时间 - 文档开始时间
        reason: "UnsupportedSecurityDoc", // 错误代码
        lastStage: {} // 同stage数据格式， 最后触发的stage 
    }
}
```
以下是错误代码Map
```javascript
{
    Unknown: 'Unknown',
    Page404: 'Page404',
    UserNotLogin: 'userNotLogin',
    // 无效链接
    InvalidLink: 'InvalidLink',
    // 需要打开密码
    NeedPassword: 'NeedPassword',
    // 需要输入修改密码
    NeedModifyPassword: 'NeedModifyPassword',
    // 密码错误
    InvalidPassword: 'InvalidPassword',
    // 协作成员已满
    SessionFull: 'SessionFull',
    // 打开失败
    OpenFailed: 'Fail',
    // 服务不可用，一般不提示给用户，静默重试
    Unavailable: "Unavailable",
    // 链接文档审核中
    FileUnderReview: "FileUnderReview",
    // 文件已被禁止访问
    LightlinkForbid: 'lightlinkForbid',
    // 您的操作权限不足
    PermissionDenied: 'PermissionDenied',
    // 已申请恢复访问
    DidResumptionVisit: 'DidResumptionVisit',
    // 打开文件时遇到错误，可能是：此文件格式和扩展名不匹配；文件已经损坏或不安全
    Fail: 'Fail',
    // 获取Token失败
    GetTokenFailed: 'GetTokenFailed',
    // 移除编辑权限
    PermissionRevoked: 'PermissionRevoked',
    // 会话过期
    SessionExpired: 'SessionExpired',
    // 暂不支持打开帐号加密文档
    UnsupportedSecurityDoc: "UnsupportedSecurityDoc",
    // 该文件大于60M，不支持预览服务
    fileTooLarge: 'fileTooLarge',
}
```




上午对表格这块接入重新调研

1、从标品几个接口代码来看，逻辑固定需要挂在页面下。 就昨晚讨论的放在子页面（解决这个问题）。
2、考虑到后期 sql数据需要使用到execl，这样导致sql插execl无法实现（标品目前只有被动接收execl数据更新，并未有实现数据主动更新功能）。

所以还是谢聪之前的方案「 插件集成WPS」这样做才合理满足。

