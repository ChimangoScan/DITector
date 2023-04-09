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

Since Docker Hub have a rate-limit 180 times in a period for every ip address, DockerCrawler provide a flexible 
choice for configuring proxies.

You can configure proxies for DockerCrawler, by simply creating file "proxyaddr.json" (default by config.json)
in `DockerCrawler/crawler/`, json file format should be structured like the example below:

```json
{
  "proxies": [
    "https://proxyaddr1.com",
    "https://proxyaddr2.com",
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