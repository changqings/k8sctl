package main

import (
	"fmt"
	"k8sctl/cronjob"
	"k8sctl/deployment"
	"log"
	"log/slog"
	"os"

	k8scrdClient "github.com/changqings/k8scrd/client"
	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name:                 "k8sctl",
		Usage:                "used for ci_cd pipeline",
		EnableBashCompletion: true,
		Version:              "v0.2.0",
		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "get k8s resources",
				Subcommands: []*cli.Command{
					{
						Name:    "deployment",
						Aliases: []string{"deploy"},
						Usage:   "get deployment unhealthy pod",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "namespace",
								Aliases:  []string{"ns"},
								Usage:    `deployment namespace,  if not set or ns=all will use check all namespace`,
								Required: false,
							},
						},
						Action: func(ctx *cli.Context) error {

							ns := ctx.String("namespace")
							cs, err := k8scrdClient.NewClient()
							if err != nil {
								return err
							}

							if ns == "" || ns == "all" {
								ns = "all"
							}
							m, err := deployment.GetUnhealthyPods(cs.KubeClient, ns)
							if err != nil {
								return err
							}

							if len(m) == 0 {
								slog.Info("not found any unhealthy pod of your cluster", "ns", ns)
								return nil
							}
							for k, v := range m {
								slog.Info("unhealthy pod", "namespace", ns, "pod_name", k, "restart_times", v)
							}
							return nil
						},
					},
				},
			},
			{
				Name:    "copy",
				Usage:   "copy k8s resources",
				Aliases: []string{"cp"},
				Subcommands: []*cli.Command{
					{
						Name:    "deployment",
						Aliases: []string{"deploy"},
						Usage:   "copy deployment from one namespace to another",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "replicas",
								Usage:    "pod's num",
								Value:    "1",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "tag",
								Usage:    "image tag of new deployment",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "deployment name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "to",
								Aliases:  []string{"t"},
								Usage:    "to namespace",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "from",
								Aliases:  []string{"f"},
								Usage:    "from namespace",
								Required: true,
							},
						},
						Action: func(ctx *cli.Context) error {
							fmt.Printf("Copy deployment and service: %s from: %s to: %s %s\n", ctx.String("name"), ctx.String("from"), ctx.String("to"), ctx.String("tag"))
							client, err := k8scrdClient.NewClient()
							if err != nil {
								log.Printf("NewClient get err: %v", err)
							}
							d := &deployment.DeploySpec{
								Client:       client.KubeClient,
								Name:         ctx.String("name"),
								Namespace:    ctx.String("from"),
								NewNamespace: ctx.String("to"),
								ImageTag:     ctx.String("tag"),
								Replicas:     int32(ctx.Int("replicas")),
							}
							if err := d.CreateNew(); err != nil {
								log.Printf("create new deploy  get err: %v", err)
								return err
							}
							return nil
						},
					},
				},
			},
			{
				Name:  "update",
				Usage: "update k8s resources",
				Subcommands: []*cli.Command{
					{
						Name:    "cronjob",
						Aliases: []string{"cron"},
						Usage:   "update k8s cronjob",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "cronjob name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "namespace",
								Aliases:  []string{"ns"},
								Usage:    "namespace",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "type",
								Aliases:  []string{"t"},
								Usage:    "type = cronjob or any other but not == api",
								Value:    "cronjob",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "labels",
								Aliases:  []string{"l"},
								Usage:    "update cronjob labels, usage: -l \"app=nginx,version=stable,cicd_env=stable...\"",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "app",
								Aliases:  []string{"a"},
								Usage:    "APP_NAME, which this cronjob belongs to",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "autocheck",
								Aliases:  []string{"auto"},
								Usage:    "auto confirm with y",
								Required: false,
							},
						},
						Action: func(ctx *cli.Context) error {
							client, err := k8scrdClient.NewClient()
							if err != nil {
								return err
							}

							c := &cronjob.CronJob{
								Client:    client.KubeClient,
								Name:      ctx.String("name"),
								Namespace: ctx.String("namespace"),
								Type:      ctx.String("type"),
								Labels:    ctx.String("labels"),
								App:       ctx.String("app"),
							}
							if c.Type == "api" {
								fmt.Println(`你输入的 --type 等于 "api" 会关联主服务 serivce,请检查!!`)
								os.Exit(1)
							}
							if ctx.String("autocheck") == "y" || ctx.String("autocheck") == "Y" {
								c.Confirm = "true"
							}

							if err := c.UpdateCronjobLabels(); err != nil {
								log.Println("Cli exec update labels err")
								return err
							}
							return nil
						},
					},
					{
						Name:    "deployment",
						Aliases: []string{"deploy"},
						Usage:   "update k8s deployment",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "deployment name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "namespace",
								Aliases:  []string{"ns"},
								Usage:    "namespace",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "type",
								Aliases:  []string{"t"},
								Usage:    "type = api/script/fe",
								Value:    "api",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "labels",
								Aliases:  []string{"l"},
								Usage:    "force update deployment labels, usage: -l \"app=nginx,version=stable,cicd_env=stable...\"",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "app",
								Aliases:  []string{"a"},
								Usage:    "app_name, when update script should have",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "autocheck",
								Aliases:  []string{"auto"},
								Usage:    "auto confirm with y",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "timeout",
								Aliases:  []string{"time"},
								Usage:    "delete tmp deployment timeout, default waiting 10s",
								Value:    "10",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "request-cpu",
								Aliases:  []string{"reqc"},
								Usage:    "request cpu 参考值: 50m",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "request-mem",
								Aliases:  []string{"reqm"},
								Usage:    "request memory 参考值: 128Mi",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "limit-cpu",
								Aliases:  []string{"limc"},
								Usage:    "limit cpu 参考值: 2000m",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "limit-mem",
								Aliases:  []string{"limm"},
								Usage:    "limit memory 参考值: 2048Mi",
								Required: false,
							},
						},
						Action: func(ctx *cli.Context) error {
							client, err := k8scrdClient.NewClient()
							if err != nil {
								return err

							}

							d := &deployment.DeploySpec{
								Client:    client.KubeClient,
								Name:      ctx.String("name"),
								Namespace: ctx.String("namespace"),
								Type:      ctx.String("type"),
								Labels:    ctx.String("labels"),
								Timtout:   int32(ctx.Int64("timeout")),
								App:       ctx.String("app"),
							}

							if ctx.String("request-cpu") != "" || ctx.String("request-mem") != "" {
								d.RequestCpu = ctx.String("request-cpu")
								d.RequestMem = ctx.String("request-mem")

								if err := d.UpdateRequests(); err != nil {
									return err
								}

								return nil
							} else if ctx.String("limit-cpu") != "" || ctx.String("limit-mem") != "" {
								d.LimitCpu = ctx.String("limit-cpu")
								d.LimitMem = ctx.String("limit-mem")

								if err := d.UpdateLimits(); err != nil {
									return err
								}
								return nil
							}

							if d.Type != "api" && d.Type != "script" && d.Type != "fe" {
								fmt.Println(`你输入的 --type 或 -t 不匹配 "api|script|fe",请检查与 --time 的区别!!`)
								os.Exit(1)
							}
							if ctx.String("autocheck") == "y" || ctx.String("autocheck") == "Y" {
								d.Confirm = "true"
							}

							if err := d.UpdateLabel(); err != nil {
								log.Printf("Cli exec update labels err")
								return err
							}
							return nil
						},
					},
				},
			},
			{
				Name:    "delete",
				Usage:   "delete k8s resources",
				Aliases: []string{"del"},
				Subcommands: []*cli.Command{
					{
						Name:    "deployment",
						Aliases: []string{"deploy"},
						Usage:   "del deployment",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "deployment name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "namespace",
								Aliases:  []string{"ns"},
								Usage:    "namespace",
								Required: true,
							},
						},
						Action: func(ctx *cli.Context) error {
							client, err := k8scrdClient.NewClient()
							if err != nil {
								return err
							}
							d := &deployment.DeploySpec{
								Client:       client.KubeClient,
								Name:         ctx.String("name"),
								NewNamespace: ctx.String("namespace"),
							}
							if ok := d.DeleteNewDeploy(); !ok {
								log.Printf("Delete deployment %s on %s failed\n", d.Name, d.NewNamespace)
								return nil
							}
							if ok := d.DeleteNewSvc(); !ok {
								log.Printf("Delete service %s on %s failed\n", d.Name, d.NewNamespace)
								return nil
							}
							return nil
						},
					},
				},
			},
		},
	}
	app.EnableBashCompletion = true
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
