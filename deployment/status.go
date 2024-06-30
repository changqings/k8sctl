package deployment

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	deployRunningThreshold     = time.Second * 100
	deployRunningCheckInterval = time.Second * 3
)

func (d *Deploy) waitForPodContainersRunning(ns, app string) error {
	end := time.Now().Add(deployRunningThreshold)
	log.Printf("等待应用 %s 的所有 pod 完成启动", app)
	for {
		<-time.NewTimer(deployRunningCheckInterval).C

		var err error
		running, err := podContainersRunning(d.Client, ns, app)
		if running {
			fmt.Println()
			return nil
		}

		if err != nil {
			println(fmt.Sprintf("Encountered an error checking for running pods: %s", err))
		}

		if time.Now().After(end) {
			return fmt.Errorf("failed to get all running containers")
		}
	}

}

func podContainersRunning(clientSet *kubernetes.Clientset, ns, app string) (bool, error) {
	// labelsSeletor :=
	pods, err := clientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		// LabelSelector: fmt.Sprintf("app=%s", app),
		LabelSelector: getPodLabels(clientSet, ns, app),
	})
	if err != nil {
		log.Printf("Pod not have label with app=%s\n", app)
		return false, err
	}
	fmt.Printf(".")

	for _, item := range pods.Items {
		for _, status := range item.Status.ContainerStatuses {
			if !status.Ready {
				return false, nil
			}
		}
	}
	return true, nil
}

func getPodLabels(clientset *kubernetes.Clientset, ns, app string) string {
	deploy, err := clientset.AppsV1().Deployments(ns).Get(context.TODO(), app, metav1.GetOptions{})

	if err != nil {
		log.Panicf("getPodLables failed ,err: %v", err)
		return ""
	}

	labelSelector := labels.SelectorFromSet(deploy.Spec.Template.Labels).String()

	return labelSelector
}

func waitDeploymentUpdate(cs *kubernetes.Clientset, ns, app string, t int) error {
	for i := 0; i <= t; i++ {
		time.Sleep(time.Second)
		if i%20 == 0 {
			d, err := cs.AppsV1().Deployments(ns).Get(context.TODO(), app, metav1.GetOptions{})
			if err != nil {
				log.Printf("Wait for update cann't get deployment, err: %v", err)
				return err
			}
			fmt.Printf(".")
			if d.Status.AvailableReplicas == d.Status.Replicas && d.Status.ReadyReplicas == d.Status.Replicas && d.Status.UpdatedReplicas == d.Status.Replicas {
				fmt.Println()
				log.Printf("成功更新 deployment = %s\n", app)
				break
			}
		}

		if i == t {
			return fmt.Errorf("等待 %d 秒 Deployment = %s 没有更新成功，程序退出!!\n请手动清理产生的临时 deployment", t, app)
		}
	}
	return nil
}
