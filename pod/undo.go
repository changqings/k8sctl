package pod

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

// sts,daemonset used a crd called controllerrevisions.apps to store history

func UndoDeploy(c *kubernetes.Clientset, name, ns string) error {
	deployGroupKind := schema.GroupKind{Group: "apps", Kind: "Deployment"}
	newDeployHistory, err := polymorphichelpers.HistoryViewerFor(deployGroupKind, c)
	if err != nil {
		return err
	}

	res, err := newDeployHistory.GetHistory(ns, name)
	if err != nil {
		return err
	}

	sortedKeys := make([]int64, 0, len(res))
	for k := range res {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	if len(sortedKeys) < 2 {
		return errors.New("history less than 2, not allow to undo")
	}

	lastRervision := sortedKeys[len(sortedKeys)-2]
	undo, ok := res[lastRervision]
	if !ok {
		return fmt.Errorf("not found history reversion %d", lastRervision)
	}
	rs, ok := undo.(*appsv1.ReplicaSet)
	if !ok {
		return errors.New("not found undo replicaSet")
	}

	deployRollbacker, err := polymorphichelpers.RollbackerFor(deployGroupKind, c)
	if err != nil {
		return err
	}

	dpObject := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}

	_, err = deployRollbacker.Rollback(&dpObject, nil, lastRervision, util.DryRunNone)
	if err != nil {
		return err
	}

	slog.Info("undo deploy success", "name", name, "namespace", ns, "rs.name", rs.Name, "revision", lastRervision)
	return nil
}
