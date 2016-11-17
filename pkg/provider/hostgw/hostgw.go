package hostgw

import (
	"fmt"
	"net"
	"os"

	log "github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
)

const (
	Namespace = "archon.kubeup.com"
)

type Handler struct{}

func server2route(s *api.Node) (*netlink.Route, error) {
	subnetStr := s.Spec.PodCIDR
	if subnetStr == "" && s.Labels != nil {
		sip, _ := s.Labels[Namespace+"/subnet-ip"]
		smask, _ := s.Labels[Namespace+"/subnet-mask"]
		if sip != "" && smask != "" {
			subnetStr = sip + "/" + smask
		}
	}
	if subnetStr == "" {
		return nil, fmt.Errorf("Unable to get subnet from node: %+v", s)
	}

	address := ""
	for _, addr := range s.Status.Addresses {
		if addr.Type == api.NodeInternalIP {
			address = addr.Address
		}
	}

	if address == "" {
		return nil, fmt.Errorf("Unable to get ip from node: %+v", s)
	}

	_, subnet, _ := net.ParseCIDR(subnetStr)
	ip := net.ParseIP(address)

	return &netlink.Route{
		Gw:  ip,
		Dst: subnet,
	}, nil
}

func routeEquals(a, b *netlink.Route) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Gw.String() == b.Gw.String() {
		if a.Dst == b.Dst {
			return true
		} else if a.Dst != nil && b.Dst != nil && a.Dst.String() == b.Dst.String() {
			return true
		}
	}
	return false
}

func (h *Handler) OnAdd(o interface{}) {
	s := o.(*api.Node)
	hostname, _ := os.Hostname()
	if s.Name == hostname {
		return
	}
	route, err := server2route(s)
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	log.Infof("Adding route %+v", route)
	if err = netlink.RouteAdd(route); err != nil {
		log.Errorf("Unable to add route: %+v\n%+v", route, err)
	}
}

func (h *Handler) OnDelete(o interface{}) {
	s, ok := o.(*api.Node)
	if !ok {
		tmp, ok := o.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Infof("Unknown event: %+v", o)
			return
		}
		s = tmp.Obj.(*api.Node)
	}
	route, err := server2route(s)
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	log.Infof("Removing route %+v", route)
	if err = netlink.RouteDel(route); err != nil {
		log.Errorf("Unable to add route: %+v\n%+v", route, err)
	}
}

func (h *Handler) OnUpdate(old, new interface{}) {
	oldNode, ok := old.(*api.Node)
	if !ok {
		return
	}
	newNode, ok := new.(*api.Node)
	if !ok {
		return
	}

	hostname, _ := os.Hostname()
	if oldNode.Name == hostname {
		return
	}

	oldRoute, err := server2route(oldNode)
	newRoute, err2 := server2route(newNode)

	if err == nil && err2 == nil && routeEquals(oldRoute, newRoute) {
		return
	}

	if err == nil {
		h.OnDelete(old)
	}
	if err2 == nil {
		h.OnAdd(new)
	}
}
