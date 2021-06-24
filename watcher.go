package nacos_registry

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"sync"
	"time"
)

type EventHandler func(event *Result)
type Watcher interface {
	Next(...EventHandler)
	Stop()
}

type Result struct {
	Action  string
	Service *Service
}

type EventType int

const (
	Create EventType = iota
	Delete
	Update
)

func (t EventType) EventType() string {
	switch t {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unknown"
	}
}

type Event struct {
	Id        string
	Type      EventType
	Timestamp time.Time
	Service   *Service
}

type nacosWatcher struct {
	n    *NacosRegistry
	next chan *Result
	exit chan bool

	sync.RWMutex
	namespace   string
	serviceName string
	nodes       map[string]*Node
}

func NewNacosWatcher(nr *NacosRegistry, serviceName string) (Watcher, error) {
	nw := &nacosWatcher{
		n:           nr,
		exit:        make(chan bool),
		next:        make(chan *Result, 10),
		serviceName: serviceName,
		nodes:       make(map[string]*Node),
	}
	instances, err := nr.client.client.SelectInstances(vo.SelectInstancesParam{ServiceName: serviceName, HealthyOnly: true})
	if err != nil && err.Error() != "instance list is empty!" {
		return nil, err
	}
	for _, instance := range instances {
		nw.nodes[instance.InstanceId] = nw.wrapRegistryNode(&instance)
	}
	param := &vo.SubscribeParam{ServiceName: serviceName, SubscribeCallback: nw.callback}
	_ = nr.client.client.Subscribe(param)
	return nw, nil
}

func (n *nacosWatcher) callback(instances []model.SubscribeService, err error) {
	if err != nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	fmt.Println(instances)
	for _, instance := range instances {
		node := n.wrapRegistryNode(&instance)
		if instance.Valid {
			if n.nodes[instance.InstanceId] == nil {
				n.next <- &Result{Action: "create", Service: n.wrapRegistryService(node)}
			} else {
				n.next <- &Result{Action: "update", Service: n.wrapRegistryService(node)}
			}
			n.nodes[instance.InstanceId] = node
		} else {
			if n.nodes[instance.InstanceId] != nil {
				n.next <- &Result{Action: "delete", Service: n.wrapRegistryService(node)}
				delete(n.nodes, instance.InstanceId)
			}
		}
	}
}

func (n *nacosWatcher) Next(handlers ...EventHandler) {
	select {
	case <-n.exit:
		return
	case r, ok := <-n.next:
		if !ok {
			return
		}
		for _, handler := range handlers {
			if r != nil {
				handler(r)
			}
		}
		return
	}
}

func (n *nacosWatcher) Stop() {
	select {
	case <-n.exit:
		return
	default:
		close(n.exit)
		param := &vo.SubscribeParam{
			ServiceName:       n.serviceName,
			SubscribeCallback: n.callback,
		}
		_ = n.n.client.client.Unsubscribe(param)
	}
}

func (n *nacosWatcher) wrapRegistryService(v *Node) *Service {
	nodes := make([]*Node, 0)
	nodes = append(nodes, v)
	return &Service{
		Name:  n.serviceName,
		Nodes: nodes,
	}
}

func (n *nacosWatcher) wrapRegistryNode(v interface{}) *Node {
	node := new(Node)
	switch ins := v.(type) {
	case *model.SubscribeService:
		node.Id = ins.InstanceId
		node.Address = n.n.joinIpPort(ins.Ip, ins.Port)
		return node
	case *model.Instance:
		node.Id = ins.InstanceId
		node.Address = n.n.joinIpPort(ins.Ip, ins.Port)
		return node
	default:
		return nil
	}
}
