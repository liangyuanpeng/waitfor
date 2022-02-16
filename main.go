/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// creates the in-cluster config
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	panic(err.Error())
	// }

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*60)
	jobInformer := informerFactory.Batch().V1().Jobs()
	informer := jobInformer.Informer()
	jobLister := jobInformer.Lister()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	stopper := make(chan struct{})
	defer close(stopper)

	informerFactory.Start(stopper)
	informerFactory.WaitForCacheSync(stopper)

	jobs, err := jobLister.Jobs("default").List(labels.Everything())
	if err != nil {
		panic(err)
	}

	go func(jobss []*v1.Job, cc chan struct{}) {
		time.Sleep(time.Second * 2)
		fmt.Println("begin send stop")
		cc <- struct{}{}
		// for idx, job := range jobss {
		// 	fmt.Printf("%d -> %s\n", idx+1, job.Name)
		// 	checkStatus(job)
		// 	cc <- struct{}{}
		// }
	}(jobs, stopper)

	fmt.Println("begin listener stopper")
	abb := <-stopper
	fmt.Println("receive:", abb)
}

func onAdd(obj interface{}) {
}

func onDelete(obj interface{}) {
}

func onUpdate(old, new interface{}) {
	// oldDeploy := old.(*v1.Job)
	newStatusJob := new.(*v1.Job)
	fmt.Println("update job:", newStatusJob.Name, newStatusJob.Name)
	checkStatus(newStatusJob)
}

func checkStatus(newStatusJob *v1.Job) {
	fmt.Println(newStatusJob.Status.Succeeded)
	if newStatusJob.Status.Succeeded > 0 {
		fmt.Println("begin stoper application")
		// stopper <- struct{}{}
	}
}
