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
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var jobname *string
var secretname *string

var namespace *string

var stopper = make(chan struct{})

func main() {

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	namespace = flag.String("namespace", "default", "which namespace of job")
	jobname = flag.String("jobname", "", "job name")
	secretname = flag.String("secret", "", "secret name")

	flag.Parse()

	if *jobname == "" && *secretname == "" {
		fmt.Println("please set the jobname with --jobname or set the secretname with --secret")
		os.Exit(-1)
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("use kubeconfig with failed |%s| and try to run with inCluster\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	runListenerForJob(clientset, stopper)
	runListenerForSecret(clientset, stopper)

	<-stopper
}

func runListenerForSecret(client *kubernetes.Clientset, c chan struct{}) {
	if *secretname == "" {
		return
	}
	informerFactory := informers.NewSharedInformerFactory(client, time.Second*10)
	secretInformer := informerFactory.Core().V1().Secrets()
	informer := secretInformer.Informer()
	secretLister := secretInformer.Lister()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdateForSecret,
		DeleteFunc: onDelete,
	})

	informerFactory.Start(stopper)
	informerFactory.WaitForCacheSync(stopper)

	secrets, err := secretLister.Secrets(*namespace).List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Wating the secret:'%s' create...\n", *secretname)
	for _, s := range secrets {
		checkStatusForSecret(s)
	}
}

func runListenerForJob(client *kubernetes.Clientset, c chan struct{}) {
	if *jobname == "" {
		return
	}
	informerFactory := informers.NewSharedInformerFactory(client, time.Second*30)
	jobInformer := informerFactory.Batch().V1().Jobs()
	informer := jobInformer.Informer()
	jobLister := jobInformer.Lister()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	informerFactory.Start(stopper)
	informerFactory.WaitForCacheSync(stopper)

	jobs, err := jobLister.Jobs(*namespace).List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Wating the job:'%s' complete...\n", *jobname)
	for _, job := range jobs {
		checkStatus(job)
	}
}

func onAdd(obj interface{}) {
}

func onDelete(obj interface{}) {
}

func onUpdate(old, new interface{}) {
	newStatusJob := new.(*v1.Job)
	// fmt.Println("update job:", newStatusJob.Name, newStatusJob.Name)
	checkStatus(newStatusJob)
}

func onUpdateForSecret(old, new interface{}) {
	newStatusSecret := new.(*corev1.Secret)
	// fmt.Println("update secret:", newStatusSecret.Name, newStatusSecret.Name)
	checkStatusForSecret(newStatusSecret)
}

func checkStatusForSecret(newStatusSecret *corev1.Secret) {
	if newStatusSecret.Name == *secretname {
		fmt.Printf("The secret:'%s' is created,exiting...\n", *secretname)
		close(stopper)
	}
}

func checkStatus(newStatusJob *v1.Job) {
	if newStatusJob.Name == *jobname {
		if newStatusJob.Status.Succeeded > 0 {
			fmt.Printf("job:%s is completed!\n", newStatusJob.Name)
			close(stopper)
		} else {
			fmt.Printf("wating for the job:%s,current status:%s\n", newStatusJob.Name, newStatusJob.Status.String())
		}
	}
}
