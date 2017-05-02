cbscfgmgr
---
# 服务端
下载[zookeeper官方镜像](https://hub.docker.com/_/zookeeper/)
docker run --name cbs-zk  -p 2181:2181 --restart always -d zookeeper
docker run -it --rm --link cbs-zk:zookeeper zookeeper zkCli.sh -server zookeeper

# 一个简单的zookeeper客户端
使用zk客户端，[参考文档](https://godoc.org/github.com/samuel/go-zookeeper/zk)
```
go get github.com/samuel/go-zookeeper
```

zk的EPHEMERAL节点则可以很方便的实现服务的注册和发现，程序启动时注册／退出时自动删除：
* 临时节点不能拥有子节点(Ephemeral nodes may not have children)
* 只支持watch本节点／子节点，不支持watch 孙节点
所以全局配置区/cbs/gcfg下只有子节点，一个子节点所有信息保存在一起

```
 >./cbscfgmgr -h
Usage of ./cbscfgmgr:
  -b int
         应用可以分配的起始节点号 
  -c string
         应用在全局配置区保存的信息：通常是ip/port/username/password
  -f string
         全局配置区信息保存到本地的文件 
  -i string
         应用保存的示例配置信息，该区只有该应用关系，全局配置区所有节点都watch
  -s string
           ZK 服务器列表，缺省是"127.0.0.1:2181"
  -t string
           应用名称, cbpapp/bmpapp/usermdb/admindb/generaldb/userpdb...
```
比如执行
```
./cbscfgmgr -b 103 -c 'abcd'  -t cbpapp
./cbscfgmgr -b 103 -c '1234'  -t cbpapp
./cbscfgmgr -b 201 -c 'billing'  -t invoicing
```
zk中保存的信息是
```
/cbs/gcfg/cbpapp:103    保存的是 abcd
/cbs/gcfg/cbpapp:104A   保存的是 1234
/cbs/gcfg/invoicing:201 保存的是 billing
```
