package deployment

import (
	"context"
	"io"
	"k8sctl/utils"
	"log"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (d *DeploySpec) BackupToLocal() {

	deploy, err := d.Client.AppsV1().Deployments(d.Namespace).Get(context.TODO(), d.Name, metav1.GetOptions{})

	if err != nil {
		log.Printf("Get deploymnet = %s.%s err: %v\n", d.Namespace, d.Name, err)
		os.Exit(1)
	}

	backupPath, err1 := utils.GetBackupPath()
	if err1 != nil {
		log.Printf("Get backup path err: %v\n", err1)
		os.Exit(1)
	}

	deployCopy := deploy.DeepCopy()
	deployCopy.APIVersion = "apps/v1"
	deployCopy.Kind = "Deployment"
	deployCopy.ManagedFields = nil

	deployYaml, err := yaml.Marshal(deployCopy)

	if err != nil {
		log.Printf("Deployment = %s.%s convert to yaml err: %v", d.Namespace, d.Name, err)
		os.Exit(1)
	}

	// if type = api , then add svc to yaml
	if d.Type == "api" || d.Type == "fe" {

		svc, err := d.Client.CoreV1().Services(d.Namespace).Get(context.TODO(), d.Name, metav1.GetOptions{})

		if err != nil {
			log.Printf("Backup service = %s.%s err: %v, please check!!, continue ...\n", d.Namespace, d.Name, err)
		} else {

			svcCopy := svc.DeepCopy()
			svcCopy.APIVersion = "v1"
			svcCopy.Kind = "Service"
			svcCopy.ManagedFields = nil
			svcBytes, err := yaml.Marshal(svcCopy)

			if err != nil {
				log.Printf("Convert service %s.%s to yaml err: %v", d.Namespace, d.Name, err)
				os.Exit(1)
			}

			// svcYaml := fmt.Sprintf("---\n" + string(svcBytes))
			deployYaml = append(deployYaml, "---\n"...)

			// can add another slice by append(slice, anotherSlice...)
			deployYaml = append(deployYaml, svcBytes...)
		}
	}

	backupFilePath := filepath.Join(*backupPath, d.Namespace+"-"+d.Name+time.Now().Format("2006-01-02-15-04-15")+".yaml")
	backupFile, err := os.Create(backupFilePath)

	if err != nil {
		log.Printf("Create %s err: %v", backupFilePath, err)
		os.Exit(1)
	}
	defer backupFile.Close()

	_, errW := backupFile.Write(deployYaml)
	if errW != nil {
		log.Printf("Write file %s err: %v\n", backupFilePath, errW)
		os.Exit(1)
	}

}

func (d *DeploySpec) NewBackupLogger() *log.Logger {
	newLog := new(log.Logger)
	backupPath, err := utils.GetBackupPath()
	if err != nil {
		log.Printf("Get backup path err: %v", err)
		os.Exit(1)
	}

	backupLogPath := filepath.Join(*backupPath, "ops.log")
	logFile, err := os.OpenFile(backupLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Panicf("Create logs file = %s err: %v", backupLogPath, err)
	}

	multiW := io.MultiWriter(os.Stdout, logFile)
	newLog.SetOutput(multiW)
	newLog.SetFlags(log.LstdFlags)
	return newLog

}
