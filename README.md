# cbscfgmgr
一个简单的zookeeper客户端
```
 >./cbscfgmgr -h
Usage of ./cbscfgmgr:
  -b int
         应用可以分配的起始节点号 
  -c string
         应用在全局配置区保存的信息：通常是ip/port/username/password
  -f string
         全局配置区信息保存的文件 
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
