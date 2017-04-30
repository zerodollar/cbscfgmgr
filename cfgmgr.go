package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/samuel/go-zookeeper/zk"
)

//CfgMgr 配置管理接口
type CfgMgr interface {
	CreateNode(nodeType string, startNum int, cfg *string) (int, error) //节点类型，开始序号，配置信息，返回分配的序号／错误信息
	FlushCfg(fileName string) error                                     //刷新 GCFGPATH中配置到指定文件中
	UpdateGlobalCfg(cfgstr *string) error                               //值
	UpdateInstCfg(cfgstr *string) error                                 //更新本应用实例配置
	Close()                                                             //释放资源
	Watch() error                                                       //监视资源变动，变动后要重新调用
}

const gCFGPATH = "/cbs/gcfg" //全局配置路径，各个节点互相感知
const gZKSEP = "/"           //zk的path分割符号
const gCBSSEP = ":"          //cbs内部拼接的符号
const gFlagFix = 0           //不是临时节点

type cbsCfgMgr struct {
	conn     *zk.Conn
	eventch  <-chan zk.Event
	nodeType string
	nodeid   int
	gPath    string
}

//NewCfgMgr 返回一个实现CfgMgr接口的对象
func NewCfgMgr(zkserver []string) (CfgMgr, error) {
	conn, eventch, err := zk.Connect(zkserver, 3*time.Second)
	if err != nil {
		return nil, err
	}
	mgr := &cbsCfgMgr{conn: conn, eventch: eventch}
	err = mgr.waitConnected()
	return mgr, err
}

func (t *cbsCfgMgr) FlushCfg(fileName string) (err error) {
	f := os.Stdout
	if fileName != "" {
		f, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660) //创建文件，如果有先删除
		if err != nil {
			log.Println(fileName, err)
			return err
		}
	}

	log.Println("Begin to output global config to ", fileName)
	nodeList, _, _ := t.conn.Children(gCFGPATH)
	for _, node := range nodeList {
		if node[0] == '_' {
			continue //跳过锁节点
		}
		val, _, _ := t.conn.Get(gCFGPATH + gZKSEP + node)
		fmt.Fprintf(f, "%s = %s\n", node, val)
	}
	log.Println("End to output global config to ", fileName)

	if fileName != "" {
		f.Close()
	}
	return nil
}

func (t *cbsCfgMgr) Close() {
	if t.conn != nil {
		t.conn.Close()
	}
}

func (t *cbsCfgMgr) UpdateGlobalCfg(cfg *string) error {
	//var tree interface{}
	//if err := json.Unmarshal([]byte(*cfg), &tree); err != nil {
	//return err
	//}
	// 可以通过 reflect 获取子节点
	//临时节点不能拥有子节点(Ephemeral nodes may not have children)
	//_, stat, err := t.conn.Get(t.gPath)   //版本肯定是0
	_, err := t.conn.Set(t.gPath, []byte(*cfg), 0)
	return err
}
func (t *cbsCfgMgr) UpdateInstCfg(cfg *string) error {
	//这个可以不用临时的，客户端退出后仍然存在， 新客户取到该id后，决定是否更新内容
	//不用临时的，可以用tree，不像gcfg中临时的，只能有一个值
	return nil
}

func (t *cbsCfgMgr) createPath(path string) error {
	pathList := strings.Split(path, gZKSEP)
	abspath := ""
	lock := zk.NewLock(t.conn, gZKSEP, zk.WorldACL(zk.PermAll))
	lock.Lock()
	defer lock.Unlock()
	for _, v := range pathList[1:] {
		abspath = abspath + gZKSEP + v
		bExist, _, err := t.conn.Exists(path)
		if bExist {
			continue
		}
		if _, err = t.conn.Create(abspath, nil, gFlagFix, zk.WorldACL(zk.PermAll)); err != nil {
			return err
		}
	}
	return nil
}
func (t *cbsCfgMgr) CreateNode(nodeType string, startNum int, cfg *string) (int, error) {
	path := gCFGPATH
	bExist, _, err := t.conn.Exists(path)
	if !bExist {
		if err = t.createPath(path); err != nil {
			return 0, err
		}
	}

	lock := zk.NewLock(t.conn, path, zk.WorldACL(zk.PermAll))
	lock.Lock()
	defer lock.Unlock()
	idlist, _, err := t.conn.Children(path)
	if err != nil {
		return 0, nil
	}
	t.nodeid = t.getIdleNode(idlist, nodeType, startNum)
	t.gPath = path + gZKSEP + nodeType + gCBSSEP + strconv.Itoa(t.nodeid)
	if _, err := t.conn.Create(t.gPath, []byte(*cfg), zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil {
		return 0, err
	}
	return t.nodeid, nil
}
func (t *cbsCfgMgr) getIdleNode(idlist []string, nodetype string, startNum int) int {
	newList := make([]int, 0, len(idlist)-1)
	for _, id := range idlist {
		if !strings.HasPrefix(id, nodetype+gCBSSEP) {
			continue
		}
		if intID, err := strconv.Atoi(strings.TrimPrefix(id, nodetype+gCBSSEP)); err == nil {
			newList = append(newList, intID)
		}
	}
	sort.Ints(newList)
	newid := startNum
	for _, v := range newList {
		if newid == v {
			newid = newid + 1
		} else if newid < v {
			break
		}
	}
	return newid
}

func (t *cbsCfgMgr) Watch() error {

	children, _, childCh, err := t.conn.ChildrenW(gCFGPATH)
	if err != nil {
		return err
	}

	for {
		select {
		case childEvent := <-childCh:
			switch childEvent.Type {
			case zk.EventNodeCreated:
				log.Println("receive znode create event, ", children, childEvent)
			case zk.EventNodeDeleted:
				log.Println("receive znode delete event, ", children, childEvent)
			case zk.EventNodeDataChanged:
				log.Println("receive znode date change event, ", children, childEvent)
			case zk.EventNodeChildrenChanged:
				log.Println("receive znode children change event ", children, childEvent)
			default:
				//log.Println("receive znode unknown event, %d\n", childEvent.Type)
				return nil
			}
		}
	}

}
func (t *cbsCfgMgr) waitConnected() error {
	for {
		isConnected := false
		select {
		case connEvent := <-t.eventch:
			if connEvent.State == zk.StateConnected {
				isConnected = true
			}
		case _ = <-time.After(time.Second * 3): // 3秒仍未连接成功则返回连接超时
			return errors.New("connect to zookeeper server timeout")
		}
		if isConnected {
			break
		}
	}
	return nil
}
