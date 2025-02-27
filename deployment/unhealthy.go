package deployment

import (
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func GetUnhealthyPods(cs *kubernetes.Clientset, ns string) (map[string]int, error) {

	if ns == "all" {
		ns = metav1.NamespaceAll
	}

	// if check map empty, use len(podMap) == 0
	podsMap := make(map[string]int)

	ds, err := cs.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{
		ResourceVersion: "0",
	})
	// if ns not found, it will return 200, and print "no deploy found on ns"
	if err != nil {
		log.Printf("Get unhealthy deployment with ns = %s error: %v", ns, err)
		return nil, err
	}

	for _, deploy := range ds.Items {

		dStat := deploy.Status
		if !(dStat.AvailableReplicas == dStat.Replicas &&
			dStat.ReadyReplicas == dStat.Replicas &&
			dStat.UpdatedReplicas == deploy.Status.Replicas) {

			pods, err := cs.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
				LabelSelector: labels.Set(deploy.Spec.Selector.MatchLabels).String(),
			})
			if err != nil {
				log.Panicf("List pod ns = %s,deployment = %s, error: %v", ns, deploy.Name, err)
				return nil, err
			}

			for _, p := range pods.Items {
				for _, v := range p.Status.ContainerStatuses {
					if !v.Ready && v.RestartCount > 30 {
						podsMap[p.Name] = int(v.RestartCount)
					}
				}
			}
		}

	}
	return podsMap, nil

}
