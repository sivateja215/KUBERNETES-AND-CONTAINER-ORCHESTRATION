// welcome-controller: a tiny Kubernetes controller written with client-go.
// Desired state: every Namespace labelled klu.edu/managed=true has a ConfigMap named "welcome".
// The controller watches Namespaces and ConfigMaps and keeps fixing any difference (reconciliation).
package main

import (
	"context" // carries timeouts/cancellation into API calls
	"flag"    // reads the -kubeconfig command-line option
	"fmt"     // prints log messages to the terminal
	"time"    // durations for resync period and worker loop

	corev1 "k8s.io/api/core/v1"                    // Go structs for core objects (Namespace, ConfigMap)
	apierrors "k8s.io/apimachinery/pkg/api/errors" // helpers: IsNotFound, IsAlreadyExists
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"  // ObjectMeta and CreateOptions
	"k8s.io/apimachinery/pkg/util/wait"            // wait.Until runs a function again and again
	"k8s.io/client-go/informers"                   // shared informer factory (list + watch + cache)
	"k8s.io/client-go/kubernetes"                  // typed client for all built-in APIs
	corelisters "k8s.io/client-go/listers/core/v1" // listers read objects from the local cache
	"k8s.io/client-go/tools/cache"                 // event handlers and WaitForCacheSync
	"k8s.io/client-go/tools/clientcmd"             // loads ~/.kube/config, exactly like kubectl
	"k8s.io/client-go/util/homedir"                // finds the user's home directory
	"k8s.io/client-go/util/workqueue"              // rate-limited queue of keys to reconcile
)

const managedLabel = "klu.edu/managed" // label key that marks a namespace as managed
const cmName = "welcome"               // name of the ConfigMap we keep in each managed namespace

func main() {
	// 1. Connect to the cluster using the same kubeconfig file kubectl uses.
	kubeconfig := flag.String("kubeconfig", homedir.HomeDir()+"/.kube/config", "path to kubeconfig")
	flag.Parse()                                                   // read command-line flags
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig) // API server address + credentials
	if err != nil {
		panic(err) // stop if the kubeconfig cannot be read
	}
	client := kubernetes.NewForConfigOrDie(config) // client used for WRITES to the API server

	// 2. Create informers: one list+watch per resource type, with a local cache.
	factory := informers.NewSharedInformerFactory(client, 10*time.Minute) // resync every 10 min
	nsInformer := factory.Core().V1().Namespaces()                        // informer for Namespaces
	cmInformer := factory.Core().V1().ConfigMaps()                        // informer for ConfigMaps

	// 3. Create the work queue. It holds namespace NAMES (keys), not whole objects.
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	// 4. Event handlers only put keys on the queue; the real work happens later in reconcile().
	nsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { queue.Add(obj.(*corev1.Namespace).Name) },    // new namespace
		UpdateFunc: func(_, obj interface{}) { queue.Add(obj.(*corev1.Namespace).Name) }, // labels may have changed
	})
	cmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) { // someone deleted a ConfigMap
			if cm, ok := obj.(*corev1.ConfigMap); ok && cm.Name == cmName {
				queue.Add(cm.Namespace) // re-check that namespace so we can heal it
			}
		},
	})

	// 5. Start the informers and wait until both caches are filled (the initial LIST is done).
	stop := make(chan struct{}) // closing this channel stops everything
	defer close(stop)
	factory.Start(stop) // starts list + watch in the background
	if !cache.WaitForCacheSync(stop, nsInformer.Informer().HasSynced, cmInformer.Informer().HasSynced) {
		panic("caches did not sync")
	}
	fmt.Println("welcome-controller started; waiting for work...")

	// 6. Worker loop: keep taking keys from the queue and reconciling them.
	lister := nsInformer.Lister() // reads Namespaces from the cache, not from the API server
	wait.Until(func() {
		for processNextItem(queue, client, lister) { // loop until the queue shuts down
		}
	}, time.Second, stop)
}

// processNextItem takes ONE key from the queue, reconciles it and handles retries.
func processNextItem(queue workqueue.TypedRateLimitingInterface[string], client kubernetes.Interface,
	lister corelisters.NamespaceLister) bool {
	name, shutdown := queue.Get() // blocks until a key is available
	if shutdown {
		return false // queue closed: stop the worker
	}
	defer queue.Done(name) // tell the queue we finished with this key

	if err := reconcile(client, lister, name); err != nil {
		fmt.Println("error reconciling", name, ":", err, "- will retry")
		queue.AddRateLimited(name) // retry later with exponential back-off
		return true
	}
	queue.Forget(name) // success: reset this key's back-off counter
	return true
}

// reconcile compares DESIRED state with ACTUAL state for one namespace and fixes the difference.
// It is idempotent: running it many times gives the same result as running it once.
func reconcile(client kubernetes.Interface, lister corelisters.NamespaceLister, name string) error {
	ns, err := lister.Get(name) // read the namespace from the local cache
	if apierrors.IsNotFound(err) {
		return nil // namespace was deleted: nothing to do
	}
	if err != nil {
		return err // unexpected error: retry
	}
	if ns.Labels[managedLabel] != "true" {
		return nil // not managed by us: leave it alone
	}

	// Desired object: a ConfigMap named "welcome" in this namespace.
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: name},
		Data:       map[string]string{"message": "Welcome to " + name + " - managed by KLU controller"},
	}
	_, err = client.CoreV1().ConfigMaps(name).Create(context.Background(), cm, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil // already there: actual state == desired state
	}
	if err == nil {
		fmt.Println("created ConfigMap", cmName, "in namespace", name) // we fixed a difference
	}
	return err
}
