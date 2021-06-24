package main

import (
	"fmt"
	nacos_registry "github.com/Kotodian/nacos-registry"
	"log"
	"net/url"
	"time"
)

func failOnError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	u := url.URL{
		Host: "139.198.171.184:8848",
	}
	config := nacos_registry.NewClientConfig("dev", 10*1000, u)
	client, err := nacos_registry.NewClientFromConfig(config)
	failOnError(err)

	registry := nacos_registry.NewRegistry(client)
	svc1 := &nacos_registry.Service{
		Name:  "ac-ocpp",
		Nodes: make([]*nacos_registry.Node, 0),
	}
	watcher1, err := registry.Watch("ac-ocpp")
	failOnError(err)
	defer watcher1.Stop()
	go func() {
		for {
			watcher1.Next(func(event *nacos_registry.Result) {
				fmt.Printf("watcher1 action: %s, node address: %s\n", event.Action, event.Service.Nodes[0].Address)
			})
		}
	}()
	svc1.Nodes = append(svc1.Nodes, &nacos_registry.Node{Address: "127.0.0.1:10086"})
	err = registry.Register(svc1)
	failOnError(err)
	svc2 := &nacos_registry.Service{
		Name:  "ac-ocpp",
		Nodes: make([]*nacos_registry.Node, 0),
	}
	svc2.Nodes = append(svc2.Nodes, &nacos_registry.Node{Address: "127.0.0.2:10086"})
	err = registry.Register(svc2)
	failOnError(err)
	time.Sleep(3 * time.Second)
	err = registry.DeRegister(svc1)
	failOnError(err)

	//svc2 := &nacos_registry.Service{
	//	Name:  "ac-ocpp-dev-02",
	//	Nodes: make([]*nacos_registry.Node, 0),
	//}
	//watcher2, err := registry.Watch("ac-ocpp-dev-02")
	//failOnError(err)
	//defer watcher2.Stop()
	//go func() {
	//	for {
	//		watcher2.Next(func(event *nacos_registry.Result) {
	//			fmt.Printf("watcher2 action: %s, node address: %s\n", event.Action, event.Service.Nodes[0].Address)
	//		})
	//	}
	//}()
	//svc2.Nodes = append(svc2.Nodes, &nacos_registry.Node{Address: "127.0.0.2:10086"})
	//err = registry.Register(svc2)
	//failOnError(err)
	//defer func() {
	//	_ = registry.DeRegister(svc2)
	//}()
	select {}
}
