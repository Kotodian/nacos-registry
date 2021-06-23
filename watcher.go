package nacos_registry

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"reflect"
	"sync"
	"time"
)

type Watcher interface {
	Next() (*Result, error)
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
	namespace     string
	services      map[string][]*Service
	cacheServices map[string][]model.SubscribeService
	Doms          []string
}

func NewNacosWatcher(nr *NacosRegistry, namespace string) (Watcher, error) {
	nw := &nacosWatcher{
		n:             nr,
		exit:          make(chan bool),
		next:          make(chan *Result, 10),
		services:      make(map[string][]*Service),
		cacheServices: make(map[string][]model.SubscribeService),
		Doms:          make([]string, 0),
		namespace:     namespace,
	}
	services, err := nr.client.client.GetAllServicesInfo(vo.GetAllServiceInfoParam{GroupName: namespace})
	if err != nil {
		return nil, err
	}
	nw.Doms = services.Doms
	for _, v := range nw.Doms {
		param := &vo.SubscribeParam{ServiceName: v, GroupName: namespace, SubscribeCallback: nw.callback}
		go func() {
			_ = nr.client.client.Subscribe(param)
		}()
	}
	return nw, nil
}

func (n *nacosWatcher) callback(services []model.SubscribeService, err error) {
	if err != nil {
		return
	}
	serviceName := services[0].ServiceName
	if n.cacheServices[serviceName] == nil {
		n.Lock()
		n.cacheServices[serviceName] = services
		n.Unlock()

		for _, service := range services {
			n.next <- &Result{Action: "create", Service: n.wrapRegistryService(&service)}
			return
		}
	} else {
		for _, subscribeService := range services {
			create := true
			for _, cacheService := range n.cacheServices[serviceName] {
				if subscribeService.InstanceId == cacheService.InstanceId {
					if !reflect.DeepEqual(subscribeService, cacheService) {
						n.next <- &Result{Action: "update", Service: n.wrapRegistryService(&subscribeService)}
					}
					create = false
				}
			}

			if create {
				n.next <- &Result{Action: "create", Service: n.wrapRegistryService(&subscribeService)}
				n.Lock()
				n.cacheServices[serviceName] = append(n.cacheServices[serviceName], subscribeService)
				n.Unlock()
				return
			}
		}
		for index, cacheService := range n.cacheServices[serviceName] {
			del := true
			for _, subscribeService := range services {
				if subscribeService.InstanceId == cacheService.InstanceId {
					del = false
				}
			}
			if del {
				n.next <- &Result{Action: "delete", Service: n.wrapRegistryService(&cacheService)}
				n.Lock()
				n.cacheServices[serviceName][index] = model.SubscribeService{}
				n.Unlock()
				return
			}
		}
	}
}
func (n *nacosWatcher) Next() (*Result, error) {
	select {
	case <-n.exit:
		return nil, nil
	case r, ok := <-n.next:
		if !ok {
			return nil, nil
		}
		return r, nil
	}
}

func (n *nacosWatcher) Stop() {
	select {
	case <-n.exit:
		return
	default:
		close(n.exit)
		if len(n.Doms) > 0 {
			for _, v := range n.Doms {
				param := &vo.SubscribeParam{
					ServiceName:       v,
					SubscribeCallback: n.callback,
				}
				_ = n.n.client.client.Unsubscribe(param)
			}
		}
	}
}

func (n *nacosWatcher) wrapRegistryService(v *model.SubscribeService) *Service {
	nodes := make([]*Node, 0)
	nodes = append(nodes, &Node{
		Id:      v.InstanceId,
		Address: n.n.joinIpPort(v.Ip, v.Port),
	})
	return &Service{
		Namespace: n.namespace,
		Name:      v.ServiceName,
		Nodes:     nodes,
	}
}
