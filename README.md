As its name shows, DockerCrawler implements a multi-thread crawler for metadata of all images from Docker Hub.
Metadata includes:

- Basic information, like namespace, repository, tags of the image, 
and the layers of each architecture of each tag, etc.
- 

*This project is the implementation for monitor module of Project DockerScanner.*

## Install

需要的数据库环境

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

### Other Configs

## Documents

程序设计思路参考`docs/dev.md`(中文文档，作者发电)