package main

import (
	"flag"
	"strings"

	"log"
)

//一个简易的客户端: ./cbscfgmgr -s zkip:port -t cbpapp -b 101 -g '{"ip":"1.2.3.4","port":"1234","user":"cbpapp","passwd":"xxx"} -f cfg.ini
//会自动申请cbpapp 101后序号，节点配置保存在cfg.ini中
func main() {
	sep := ","
	zkServer := flag.String("s", "127.0.0.1:2181", "ZK server list, Comma separated string,ip1:port1,ip2:port2")
	nodeType := flag.String("t", "cbpapp", "cbpapp/bmpapp/usermdb/admindb/generaldb/userpdb...")
	startNum := flag.Int("b", 101, "begin node id")
	gcfg := flag.String("c", "{}", "global config json")
	icfg := flag.String("i", "{}", "node config json //TODO")
	outFile := flag.String("f", "", "output the global config to this file")
	flag.Parse()

	zkmgr, err := NewCfgMgr(strings.Split(*zkServer, sep))
	must(err)
	defer zkmgr.Close()

	nodeID, err := zkmgr.CreateNode(*nodeType, *startNum, gcfg)
	must(err)

	err = zkmgr.UpdateInstCfg(icfg)
	must(err)

	for {
		zkmgr.FlushCfg(*outFile)
		log.Printf("begin to watch %s:%d\n", *nodeType, nodeID)
		zkmgr.Watch()
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
