# Allinssl-AutoCdnFly-Plugin
这是个AllinSSL插件，监控证书状态并自动更新部署到CdnFly系统。
## 说明
- 此插件适配“失控的防御系统(scdn.io)”的cdnfly系统，不确保cdnfly系统标准接口可用。
- 如果你使用的是cdnfly系统标准接口，可根据[cdnfly官方文档](https://doc.cdnfly.com/shiyongjieshao.html)进行适配。
## 使用
1. 将此插件编译的二进制可执行文件放置于AllinSSL主程序运行目录下的`plugins`目录下
2. 进入AllinSSL后台->授权API管理->添加授权API
3. 类型选择`插件`，插件名称选择`AutoCDNfly`，根据输入框提示依次填写`基础API接口`、`api_key`、`api_secret`，然后确认即可
4. 然后你就可以在`自动化部署`的`部署节点`选择`AutoCDNfly`插件了
## 注意
- `基础API接口`填写例子：https://user.cdn1.vip/v1
- `api_key`和`api_secret`在CdnFly系统后台的`账户中心`->`API密钥`中获取
- 如果你是通过宝塔面板Docker一键部署的AllinSSL系统，则需要在宝塔面板后台的->`Docker`->`容器编排`->你的AllinSSL容器编排->`配置文件`中的`docker-compose文件内容`添加一条文件目录映射才能安装此插件：

`volumes:`配置项下添加`- ${APP_PATH}/plugins:/www/allinssl/plugins`

完整volumes配置：
``` docker-compose
volumes:
  - ${APP_PATH}/plugins:/www/allinssl/plugins
  - ${APP_PATH}/data:/www/allinssl/data
  - ${APP_PATH}/logs:/www/allinssl/logs
```

保存重建后（此操作不会清空原有数据），将此插件上传到`/www/dk_project/dk_app/allinssl/allinssl/plugins`目录下即可

## AutoCdnFly 执行逻辑
- CdnFly系统不存在当前申请证书对应域名的站点或域名 -> 过滤
- CdnFly系统内的证书过期时间大于30天 -> 过滤
- CdnFly系统和当前申请的证书匹配成功且满足以下表的`触发场景`即执行对应操作

|operate 值|触发场景|行为|
|----|----|----|
|create|站点未绑定证书 / 绑定的证书ID 不存在|新建证书 + 绑定到站点|
|update|站点有证书，剩余有效期 < 30 天|直接更新原有证书内容|
|replace|站点有证书、证书未过期、域名不匹配|新建证书 + 绑定（替换旧证书）|
