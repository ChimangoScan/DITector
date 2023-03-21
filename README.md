### This project is the implementation for monitor module of Project DockerScanner.

### 配置代理池

如果想配置Docker Hub Crawler代理池，需要在crawler文件夹下创建proxyaddr.json文件，文件格式可以参考：

```json
{
  "proxies": [
    "https://proxyaddr1.com",
    "https://proxyaddr2.com",
    "..."
  ]
}
```