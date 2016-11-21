// Package main is the entry point of hostroutes
package main

import (
	goflag "flag"
	log "github.com/golang/glog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	flag "github.com/spf13/pflag"
	"kubeup.com/hostroutes/pkg/provider/hostgw"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	overrides := &clientcmd.ConfigOverrides{}
	overrideFlags := clientcmd.RecommendedConfigOverrideFlags("")
	clientcmd.BindOverrideFlags(overrides, flags, overrideFlags)

	// Parse flags
	resync := flags.Int("resync", 30, "Resync period in seconds")
	incluster := flags.Bool("in-cluster", false, "If this in run inside a pod")
	profile := flags.Bool("profile", false, "Enable profiling")
	address := flags.String("profile_host", "localhost", "Profiling server host")
	port := flags.Int("profile_port", 9801, "Profiling server port")
	test := flags.Bool("test", false, "Dry-run. To test if the binary is complete")
	flags.AddGoFlagSet(goflag.CommandLine)
	flags.Parse(os.Args)

	if *test {
		return
	}

	// Set up profiling server
	if *profile {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

			server := &http.Server{
				Addr:    net.JoinHostPort(*address, strconv.Itoa(*port)),
				Handler: mux,
			}
			log.Fatalf("%+v", server.ListenAndServe())
		}()
	}

	// Create kubeconfig
	var (
		clientConfig *rest.Config
		err          error
	)
	if *incluster {
		clientConfig, err = rest.InClusterConfig()
	} else {
		kubeconfig := clientcmd.NewDefaultClientConfig(*clientcmdapi.NewConfig(), overrides)
		clientConfig, err = kubeconfig.ClientConfig()
	}

	if err != nil {
		log.Fatalf("Unable to create config: %+v", err)
	}

	// Create kubeclient
	k8s, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatalf("Invalid api configuration: %+v", err)
	}

	// Create listwatch instance
	listwatch := cache.NewListWatchFromClient(k8s.Core().RESTClient(), "nodes", api.NamespaceAll, fields.Everything())

	// Create informer
	_, informer := cache.NewInformer(listwatch, &v1.Node{}, time.Second*(time.Duration)(*resync), &hostgw.Handler{})

	// Handle signals (optional)
	// Start watching
	log.Infof("Hostroutes starting...")
	_, err = k8s.Core().Nodes().List(v1.ListOptions{})
	if err != nil {
		log.Fatalf("Unable to connect k8s master. Aborting")
	}
	informer.Run(wait.NeverStop)
	log.Infof("Done")
}
