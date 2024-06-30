package deployment

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var dryRun []string

func (d *Deploy) getSvc(name, ns string) *corev1.Service {
	svc, err := d.Client.CoreV1().Services(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		log.Printf("INFO: Service = %s, namespace = %s not found.\n", name, ns)
		return nil
	}
	return svc

}

func (d *Deploy) DeleteNewSvc() bool {
	dryRun = append(dryRun, "All")

	err := d.Client.CoreV1().Services(d.NewNamespace).Delete(context.TODO(), d.Name, metav1.DeleteOptions{
		DryRun: dryRun,
	})
	if err != nil {
		log.Printf("DryRun delete svc = %s, namespace = %s error: %s\n", d.NewNamespace, d.Name, err)
		return false
	}

	log.Printf("DryRun delete svc = %s, namespace = %s successfully.\n", d.Name, d.NewNamespace)
	_ = d.Client.CoreV1().Services(d.NewNamespace).Delete(context.TODO(), d.Name, metav1.DeleteOptions{})

	log.Printf("Delete svc = %s, namespace = %s successfully.\n", d.Name, d.NewNamespace)
	return true
}

func (d *Deploy) createNewSvc(oriService *corev1.Service) *corev1.Service {

	oriServiceDeep := oriService.DeepCopy()
	oriServiceDeep.Namespace = d.NewNamespace
	oriServiceDeep.Spec.Type = corev1.ServiceTypeClusterIP
	oriServiceDeep.ResourceVersion = ""
	oriServiceDeep.Spec.ExternalTrafficPolicy = ""
	oriServiceDeep.Spec.ClusterIP = ""
	oriServiceDeep.Spec.ClusterIPs = nil

	for k, v := range oriServiceDeep.Spec.Ports {
		if v.NodePort != 0 {
			oriServiceDeep.Spec.Ports[k].NodePort = 0
		}
	}

	newSvc, err := d.Client.CoreV1().Services(d.NewNamespace).Create(context.TODO(), oriServiceDeep, metav1.CreateOptions{})
	if err != nil {
		log.Panicf("Create service = %s, namespace = %s err %s\n", d.Name, d.NewNamespace, err)
	}

	log.Printf("Create service = %s, namespace = %s complete.\n", d.Name, d.NewNamespace)

	return newSvc

}
