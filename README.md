As its name shows, DockerCrawler implements a multi-thread crawler for metadata of all images from Docker Hub.
Metadata includes:

- Basic information, like namespace, repository, tags of the image, 
and the layers of each architecture of each tag, etc.
- 

*This project is the implementation for monitor module of Project DockerScanner.*

## Install

配置mysql数据库环境，并为mysql数据库创建新用户docker，密码docker。

## Usage
### Basic Usage

After installation of DockerCrawler and other environment requirements (such as mysql), you can begin 

### Proxies

Docker Hub有访问频率限制：每个IP地址 180次/某时间段, DockerCrawler允许自己配置代理池来防止IP访问过快被禁止访问。

如果您已经有一个稳定的静态IP代理池，那么可以在`crawler/`路径下创建`proxyaddr.json`文件（具体路径可以通过config.json设置），按照如下格式组织内容，即可配置代理池。

```json
{
  "proxies": [
    "proxyaddr1.com",
    "proxyaddr2.com",
    "..."
  ]
}
```

Besides, we originally use proxies from [kuaidaili](https://www.kuaidaili.com/) and implement automatic proxy
updater to monitor the life of every proxy-ip and substitude those ips to be out-of-life with new ones.

If you decide to use kuaidaili as proxy-ip source too, just create a file named "secret.json" under the directory
crawler/, and the content should be:

```json
{
  "secret_id": "",
  "secret_key": ""
}
```

### Other Configs

## Documents

程序设计思路参考`docs/dev.md`(中文文档，作者发电)