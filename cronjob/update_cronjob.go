package cronjob

import (
	"bufio"
	"context"
	"fmt"
	"k8sctl/utils"
	"log"
	"os"
	"reflect"
	"time"

	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type CronJob struct {
	Client    *kubernetes.Clientset
	Name      string
	Namespace string
	Labels    string
	Confirm   string
	App       string
	Type      string
}

func NewClient(client *kubernetes.Clientset) *CronJob {

	return &CronJob{
		Client: client,
	}
}

func (c *CronJob) UpdateCronjobLabels() error {
	newLabels := make(map[string]string)

	if c.Labels == "" {
		// stable cronjob labels
		newLabels["app"] = c.App
		newLabels["name"] = c.Name
		newLabels["type"] = c.Type
		newLabels["cicd_env"] = "stable"
		newLabels["version"] = "stable"

	} else {
		newLabels = utils.StringToMap(c.Labels)
		if newLabels == nil {
			log.Printf(`解析 Lables = %s 失败，请按格式"app=xx,version=xx"进行传值`, c.Labels)
			os.Exit(1)
		}
	}

	if c.Type == "api" {
		log.Printf("Cronjob = %s in namespace = %s,不能使用标签 type=api", c.Name, c.Namespace)
		os.Exit(1)
	}

	oriCronjob := c.getCronjob()
	if oriCronjob == nil {
		log.Println("获取 cronjob 失败, 请检查")
		os.Exit(1)
	}

	log.Printf("开始对 %s.%s 进行标签替换\n", c.Namespace, c.Name)
	fmt.Println("请确认信息: ")

	// logger set No Ldate | Ltime
	log.SetFlags(0)
	log.Printf("\n-----替换 Cronjob = %s 标签-----\n", c.Name)
	log.Println("^^ 原标签为：")
	for k, v := range oriCronjob.ObjectMeta.Labels {
		log.Printf("%s=%s\n", k, v)
	}
	log.Printf("\n$$ 替换的标签为：\n")
	for k, v := range newLabels {
		log.Printf("%s=%s\n", k, v)
	}

	// logger reset LstdFlags = 3
	log.SetFlags(3)

	fmt.Println()
	if reflect.DeepEqual(oriCronjob.ObjectMeta.Labels, newLabels) && reflect.
		DeepEqual(oriCronjob.Spec.JobTemplate.ObjectMeta.Labels, newLabels) && reflect.
		DeepEqual(oriCronjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels, newLabels) {
		log.Printf("要修改的 Cronjob 的标签和原标签完全一样，程序退出！！")
		os.Exit(0)
	}
	log.Printf("是否确认执行？输入 [ y|Y ] 继续, Ctrl^C 退出（回车确认输入）: ")

	if c.Confirm == "" {
		var execConfirm string
		for {
			fmt.Printf("请输入确认[ y|Y ]: ")
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

	//Force update cronjob with new labels

	log.Printf("开始修改标签 cronjob = %s, 请稍等 ...", c.Name)
	time.Sleep(1 * time.Second)

	var tryTimes = 3
	var timeSleep = time.Second
	for i := 1; i <= tryTimes; i++ {
		oriCronjob := c.getCronjob()
		if oriCronjob.ObjectMeta.Labels != nil {
			oriCronjob.ObjectMeta.Labels = nil
			oriCronjob.Spec.JobTemplate.Labels = nil
			oriCronjob.Spec.JobTemplate.Spec.Template.Labels = nil
		}
		oriCronjob.ObjectMeta.Labels = newLabels
		oriCronjob.Spec.JobTemplate.Labels = newLabels
		oriCronjob.Spec.JobTemplate.Spec.Template.Labels = newLabels
		_, err1 := c.Client.BatchV1beta1().CronJobs(c.Namespace).Update(context.Background(), oriCronjob, metav1.UpdateOptions{})
		if err1 == nil {
			break
		}
		log.Printf("第 %d 次尝试更新标签失败，将在 1s 后重试", i)
		time.Sleep(timeSleep)
		if i == tryTimes {
			log.Panicf("重试 %d 后，更新 cronjob = %s.%s 失败，请联系运维人员。\n", tryTimes, c.Name, c.Namespace)
			os.Exit(1)
		}
	}

	log.Println("修改标签完成")
	return nil
}

func (c *CronJob) getCronjob() *v1beta1.CronJob {

	cronjob, err := c.Client.BatchV1beta1().CronJobs(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("获取 cronjob = %s in namespace = %s err: %v\n", c.Name, c.Namespace, err)
		return nil
	}
	return cronjob

}
