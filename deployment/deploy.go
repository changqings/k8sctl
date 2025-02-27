package deployment

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"k8sctl/utils"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeploySpec struct {
	Client       *kubernetes.Clientset
	Name         string
	Namespace    string
	NewNamespace string
	ImageTag     string
	Replicas     int32
	Type         string
	Labels       string
	Confirm      string
	Timtout      int32
	App          string
	RequestCpu   string
	RequestMem   string
	LimitCpu     string
	LimitMem     string
}

// var srcDeploy = &appsv1.Deployment{}
// var svc = &corev1.Service{}

func NewDeploy(client *kubernetes.Clientset) *DeploySpec {

	return &DeploySpec{
		Client: client,
	}
}

func (d *DeploySpec) UpdateLimits() error {
	deploy, err := d.Client.AppsV1().Deployments(d.Namespace).Get(context.Background(), d.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containers := deploy.Spec.Template.Spec.Containers
	for i := 0; i < len(containers); i++ {
		containers[i].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(d.LimitCpu),
			corev1.ResourceMemory: resource.MustParse(d.LimitMem),
		}
	}

	_, err = d.Client.AppsV1().Deployments(d.Namespace).Update(context.Background(), deploy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	log.Printf("Update deployment = %s.%s resources.limits success.\n", d.Name, d.Namespace)

	return nil

}
func (d *DeploySpec) UpdateRequests() error {
	deploy, err := d.Client.AppsV1().Deployments(d.Namespace).Get(context.Background(), d.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containers := deploy.Spec.Template.Spec.Containers
	for i := 0; i < len(containers); i++ {
		containers[i].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(d.RequestCpu),
			corev1.ResourceMemory: resource.MustParse(d.RequestMem),
		}
	}

	_, err = d.Client.AppsV1().Deployments(d.Namespace).Update(context.Background(), deploy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	log.Printf("Update deployment = %s.%s resources.requests success.\n", d.Name, d.Namespace)

	return nil

}

func (d *DeploySpec) UpdateLabel() error {
	log := d.NewBackupLogger()
	svc := &corev1.Service{}

	deployUpdateLabels := make(map[string]string)
	serviceUpdateLabels := make(map[string]string)

	if len(d.Labels) == 0 {
		// stable deployment labels
		deployUpdateLabels["app"] = d.Name
		deployUpdateLabels["name"] = d.Name
		deployUpdateLabels["type"] = d.Type
		if d.App != "" {
			deployUpdateLabels["app"] = d.App
		}
		deployUpdateLabels["cicd_env"] = "stable"
		deployUpdateLabels["version"] = "stable"

		// stable deployment labels
		serviceUpdateLabels["app"] = d.Name
		if d.App != "" {
			serviceUpdateLabels["app"] = d.App
		}
		serviceUpdateLabels["name"] = d.Name
		serviceUpdateLabels["type"] = "api"

	} else {
		deployUpdateLabels = utils.StringToMap(d.Labels)
		serviceUpdateLabels = utils.StringToMap(d.Labels)
		if deployUpdateLabels == nil {
			log.Printf("解析 Lables = %s 失败，请按格式\"app=xx,version=xx\"进行传值", d.Labels)
			return errors.New("解析 Lables 失败, 请按格式\"app=xx,version=xx\"进行传值")
		}
		if d.Type == "fe" {
			delete(serviceUpdateLabels, "version")
		} else {
			delete(serviceUpdateLabels, "cicd_env")
			delete(serviceUpdateLabels, "version")
		}
	}

	if d.Type == "api" || d.Type == "fe" {
		svc = d.GetSvc(d.Name, d.Namespace)
		if svc == nil {
			log.Printf("Service = %s not found in namespace = %s, 请检查是否在没有 service 的情况下使用了 --type=api", d.Name, d.Namespace)
			return errors.New("service not found")
		}
	}

	oriDeployment := d.getDeploy(d.Name, d.Namespace)

	if oriDeployment == nil {
		log.Printf("Deployment = %s not found in namesapce = %s", d.Name, d.Namespace)
		return errors.New("deployment not found")
	}

	log.Printf("开始对 %s.%s 进行标签替换\n, 请确认信息：", d.Namespace, d.Name)

	// logger set No Ldate | Ltime
	log.SetFlags(0)
	log.Printf("\n-----替换 Deployment = %s 标签-----\n", d.Name)
	log.Println("^^ 原标签为：")
	for k, v := range oriDeployment.Spec.Template.Labels {
		log.Printf("%s=%s\n", k, v)
	}
	log.Printf("\n$$ 替换的标签为：\n")
	for k, v := range deployUpdateLabels {
		log.Printf("%s=%s\n", k, v)
	}

	if d.Type == "api" || d.Type == "fe" {
		log.Printf("\n-----替换 Service = %s 标签-----\n", d.Name)
		log.Println("^^ 原标签为：")
		for k, v := range svc.Spec.Selector {
			log.Printf("%s=%s\n", k, v)
		}
		log.Print("\n$$ 替换到标签：\n")
		for k, v := range serviceUpdateLabels {
			log.Printf("%s=%s\n", k, v)
		}
	}
	// logger reset LstdFlags = 3
	log.SetFlags(3)

	fmt.Println()
	if reflect.DeepEqual(oriDeployment.Spec.Template.Labels, deployUpdateLabels) {
		if d.Type == "api" || d.Type == "fe" && reflect.DeepEqual(svc.Spec.Selector, serviceUpdateLabels) {
			log.Printf("要修改的 Deployment 和 Service 的标签和原标签完全一样，程序退出！！")
			return nil
		} else if d.Type == "api" || d.Type == "fe" && !reflect.DeepEqual(svc.Spec.Selector, serviceUpdateLabels) {
			log.Printf("要修改的 Deployment 的标签一样,但是 Service 的 selector 标签不一样，继续运行...")
		} else {
			log.Printf("要修改的 Deployment 的标签和原标签完全一样，程序退出！！")
			return nil
		}
	}
	log.Printf("是否确认执行? 请输入 [ y|Y ], Ctrl^C 退出（回车确认输入）: ")

	if d.Confirm == "" {
		var execConfirm string
		for {
			fmt.Printf("请输入确认 [ y|Y ]: ")
			stdin := bufio.NewReader(os.Stdin)
			_, err := fmt.Fscan(stdin, &execConfirm)
			stdin.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				log.Printf("你输入的字符 = %v, 请重新输入!!", execConfirm)
				continue
			}
			if execConfirm != "y" && execConfirm != "Y" {
				log.Printf("你输入的字符 = %v, 请重新输入!!", execConfirm)
				continue
			} else {
				break
			}
		}
	}

	// backup deployment.yaml in $HOME/.kube/k8sctl-backups/
	d.BackupToLocal()

	// Create tmp Deployment
	tmpDeployment := d.createTmpDeploy(oriDeployment)
	log.Printf("创建临时 Deployment = %s-tmp, 请稍等 ...", d.Name)

	if tmpDeployment == nil {
		log.Printf("Create deployment = %s failed.", oriDeployment.Name)
		os.Exit(1)
	}

	if err := WaitDeploymentUpdate(d.Client, tmpDeployment.Namespace, tmpDeployment.Name, 180); err != nil {
		log.Printf("Tmp deployment = %s started failed, please check.", tmpDeployment.Name)
		os.Exit(1)
	}

	// Force update deployment with new labels

	newDeploy := oriDeployment.DeepCopy()
	newDeploy.ObjectMeta.Labels = deployUpdateLabels
	newDeploy.Spec.Selector.MatchLabels = deployUpdateLabels
	newDeploy.Spec.Template.ObjectMeta.Labels = deployUpdateLabels
	newDeploy.ObjectMeta.UID = ""
	newDeploy.ObjectMeta.ResourceVersion = ""

	log.Printf("开始修改标签 Deployment = %s, 大约需要 2 分钟，请稍等 ...", d.Name)
	time.Sleep(1 * time.Second)

	var graceTimeout int64 = 8
	err := d.Client.AppsV1().Deployments(d.Namespace).Delete(context.TODO(), oriDeployment.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &graceTimeout,
	})
	if err != nil {
		log.Printf("Delete deployment = %s.%s failed, err = %v", d.Namespace, d.Name, err)
		os.Exit(1)
	}

	newDeploy = d.addPrestop(newDeploy)
	if newDeploy == nil {
		log.Printf("Create newDeployment with preStop err, please check")
		return nil
	}

	time.Sleep(1 * time.Second)
	_, err1 := d.Client.AppsV1().Deployments(d.Namespace).Create(context.TODO(), newDeploy, metav1.CreateOptions{})
	if err1 != nil {
		log.Printf("Force update deployment = %s.%s  labels failed, err = %v", d.Namespace, d.Name, err1)
		os.Exit(1)
	}

	errWait := WaitDeploymentUpdate(d.Client, d.Namespace, d.Name, 180)
	if errWait != nil {
		log.Printf("Wait pod Running err: %v", errWait)
	}
	log.Printf("修改标签完成 Deployment = %s", d.Name)

	// Update svc selector lables
	if d.Type == "api" || d.Type == "fe" {
		time.Sleep(3 * time.Second)
		log.Printf("开始更新 Service 标签 = %s", d.Name)

		svc.ObjectMeta.Labels = nil
		svc.Spec.Selector = nil
		svc.ObjectMeta.Labels = serviceUpdateLabels
		svc.Spec.Selector = serviceUpdateLabels

		_, err2 := d.Client.CoreV1().Services(d.Namespace).Update(context.TODO(), svc, metav1.UpdateOptions{})

		if err2 != nil {
			log.Printf("Force update service = %s.%s labels failed, err = %v", d.Namespace, d.Name, err2)
			os.Exit(1)
		}
		log.Printf("标签更新完成 Service = %s ", d.Name)

	} else {
		log.Printf(`你输入的 Type = %s, 类型不是[ api|fe ],跳过更新Service`, d.Type)
	}

	// Delete tmp deployment
	log.Printf("开始删除临时应用 Deployment = %s.%s-tmp\n请检查后, 确认执行? 确认请输入 [ y|Y ], Ctrl^C 退出（回车确认输入）: ", d.Namespace, d.Name)
	if d.Confirm == "" {
		var delConfirm string
		for {
			fmt.Print("请输入确认[ y|Y ]: ")
			stdin := bufio.NewReader(os.Stdin)
			_, err := fmt.Fscan(stdin, &delConfirm)
			stdin.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				log.Printf("你输入的字符 = %v, 请重新输入!!", delConfirm)
				continue
			}
			if delConfirm != "y" && delConfirm != "Y" {
				log.Printf("你输入的字符 = %v, 请重新输入!!", delConfirm)
				continue
			} else {
				break
			}

		}
	}

	log.Printf("开始删除临时应用 Deployment = %s-tmp\n删除将在 10 秒后执行, 等待服务 endpoint 列表同步\n", d.Name)

	if d.Timtout > 0 {
		for i := 0; i < int(d.Timtout); i++ {
			if i%3 == 0 {
				fmt.Printf(".")
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		log.Printf("更新设置 timeout = 0 , 删除临时应用立即执行")
	}
	fmt.Println()

	if err = d.Client.AppsV1().Deployments(d.Namespace).Delete(context.TODO(), fmt.Sprintf("%s-tmp", d.Name), metav1.DeleteOptions{
		GracePeriodSeconds: &graceTimeout,
	}); err != nil {
		log.Panicf("Delete deployment = %s.%s-tmp failed, err = %v", d.Namespace, d.Name, err)
		os.Exit(1)
	}
	log.Printf("成功删除临时 deployment = %s-tmp\n应用标签替换完成 deployment = %s", d.Name, d.Name)

	return nil
}

func (d *DeploySpec) CreateNew() error {
	// copy service
	log.Println("Copy Service ...")
	oriService := d.GetSvc(d.Name, d.Namespace)

	if oriService == nil {
		return fmt.Errorf("在命名空间= %s 没有发现服务= %s, 请先部署到命名空间 %s,再重试", d.Namespace, d.Name, d.Namespace)
	}
	log.Printf("Service = %s, namespace = %s has found. Continue ...", d.Name, d.Namespace)

	dstService := d.GetSvc(d.Name, d.NewNamespace)

	if dstService != nil {
		log.Printf("Service = %s, namespace = %s has found. Recreating it ...", d.Name, d.NewNamespace)
		if ok := d.DeleteNewSvc(); !ok {
			log.Println("Delete service failed")
		}
	}
	d.CreateNewSvc(oriService)

	// copy deployment
	log.Println("Copy Deployment ...")
	srcDeploy := d.getDeploy(d.Name, d.Namespace)

	if srcDeploy == nil {
		return fmt.Errorf("在命名空间= %s 没有发现服务= %s, 请先部署到命名空间 %s,再重试", d.Namespace, d.Name, d.Namespace)
	}
	log.Printf("Deployment = %s, namespace = %s has found. Continue ...", d.Name, d.Namespace)

	dstDeploy := d.getDeploy(d.Name, d.NewNamespace)

	if dstDeploy != nil {
		log.Printf("Deployment = %s, namespace = %s has found. Recreating it ...", d.Name, d.NewNamespace)
		if ok := d.DeleteNewDeploy(); !ok {
			log.Println("Delete deployment failed")
		}
	}
	d.createNewDeploy(srcDeploy)

	// waitfor deployment
	err := WaitDeploymentUpdate(d.Client, d.NewNamespace, d.Name, 180)

	if err != nil {
		log.Printf("wait for pod running err: %s\n", err)
		return err
	}
	log.Println("Pod running successfully!")

	return nil

}
func (d *DeploySpec) getDeploy(name, ns string) *appsv1.Deployment {

	deploy, err := d.Client.AppsV1().Deployments(ns).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		log.Printf("INFO: deployment = %s, namespace = %s not found.\n", name, ns)
		return nil
	}

	return deploy

}

func (d *DeploySpec) DeleteNewDeploy() bool {
	dryRun = append(dryRun, "All")

	err := d.Client.AppsV1().Deployments(d.NewNamespace).Delete(context.TODO(), d.Name, metav1.DeleteOptions{
		DryRun: dryRun,
	})

	if err != nil {
		log.Printf("Dryrun delete deployment = %s, namespace = %s err: %s\n", d.Name, d.NewNamespace, err)
		return false
	}
	log.Printf("Dryrun delete deployment = %s, namespace = %s successfully.\n", d.Name, d.NewNamespace)

	var graceTimeout int64 = 40
	_ = d.Client.AppsV1().Deployments(d.NewNamespace).Delete(context.TODO(), d.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &graceTimeout,
	})
	log.Printf("Delete deployment = %s, namespace = %s successfully.\n", d.Name, d.NewNamespace)

	return true

}

func (d *DeploySpec) createNewDeploy(oriDeploy *appsv1.Deployment) *appsv1.Deployment {

	oriDeployDeep := oriDeploy.DeepCopy()
	oriDeployDeep.Namespace = d.NewNamespace
	oriDeployDeep.ResourceVersion = ""
	oriDeployDeep.Spec.Replicas = &d.Replicas

	if d.ImageTag != "" {
		image := oriDeployDeep.Spec.Template.Spec.Containers[0].Image
		s := strings.Split(image, ":")
		if len(s) == 2 {
			s[1] = d.ImageTag
			oriDeployDeep.Spec.Template.Spec.Containers[0].Image = strings.Join(s, ":")
		}
	}

	newDeploy, err := d.Client.AppsV1().Deployments(d.NewNamespace).Create(context.TODO(), oriDeployDeep, metav1.CreateOptions{})

	if err != nil {
		log.Panicf("Create deployment = %s, namespace = %s err %s\n", d.Name, d.NewNamespace, err)
	}
	log.Printf("Create deployment = %s, namesapce = %s complete.\n", d.Name, d.NewNamespace)

	return newDeploy
}

func (d *DeploySpec) createTmpDeploy(oriDeploy *appsv1.Deployment) *appsv1.Deployment {
	oriDeployDeep := oriDeploy.DeepCopy()
	oriDeployDeep.Name = d.Name + "-tmp"
	oriDeployDeep.ResourceVersion = ""

	deploy := d.addPrestop(oriDeployDeep)
	if deploy == nil {
		log.Printf("Create Tmp Deployment with preStop err, please check")
		return nil
	}

	tmpDeploy, err := d.Client.AppsV1().Deployments(d.Namespace).Create(context.TODO(), deploy, metav1.CreateOptions{})
	if err != nil {
		log.Printf("创建 deployment = %s, namespace = %s 将执行重建临时 deployment, err = %v", oriDeployDeep.Name, d.Namespace, err)
		_ = d.Client.AppsV1().Deployments(d.Namespace).Delete(context.TODO(), oriDeployDeep.Name, metav1.DeleteOptions{})
		tmpDeploy, err1 := d.Client.AppsV1().Deployments(d.Namespace).Create(context.TODO(), oriDeployDeep, metav1.CreateOptions{})
		if err1 != nil {
			log.Printf("重建临时 Deployment = %s.%s 失败，程序退出！！请检查", oriDeployDeep.Name, d.Name)
			return nil
		}
		return tmpDeploy
	}
	return tmpDeploy
}

func (d *DeploySpec) addPrestop(deploy *appsv1.Deployment) *appsv1.Deployment {

	preStopCommmand := &corev1.Lifecycle{}
	appContainerIndex := 0

	if deploy.Spec.Template.Spec.Containers != nil {
		for index, c := range deploy.Spec.Template.Spec.Containers {
			if c.Name == "app" {
				appContainerIndex = index
				if c.Lifecycle == nil {
					preStopCommmand = &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 5"},
							},
						},
					}
					break
				}
				if c.Lifecycle != nil && c.Lifecycle.PostStart != nil {
					preStopCommmand = &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 5"},
							},
						},
						PostStart: c.Lifecycle.PostStart,
					}
				}
			}
		}
		deploy.Spec.Template.Spec.Containers[appContainerIndex].Lifecycle = preStopCommmand
	}
	return deploy

}
