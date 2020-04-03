package cron_hpa

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/robfig/cron"
	restclient "k8s.io/client-go/rest"
	"time"
)

const CronHpaConfigKey = "qsun.tencent.com/cronhpa"

var CurNs = GetCurrentNamesace()

type Controller struct {
	cs *kubernetes.Clientset

	// key: hash of deployment name and config string
	crons map[uint32]*cron.Cron
	hpas *v2beta2.HorizontalPodAutoscalerList
}

func NewController() (*Controller, error) {
	config, err := restclient.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot create in-cluster config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create client set: %w", err)
	}

	result := &Controller{
		cs:    cs,
		crons: make(map[uint32]*cron.Cron),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			<- ticker.C
			result.hpas, err = result.cs.AutoscalingV2beta2().HorizontalPodAutoscalers(CurNs).List(v1.ListOptions{})
			if err != nil {
				logrus.Errorf("Cannot list hpa: %v", err.Error())
			}
			err = result.ScanAndCrontab()
			if err != nil {
				logrus.Errorf("%v", err.Error())
			}
		}
	}()

	return result, nil
}

// ScanAndCrontab Scans all deployments' annotations in current namespace and initialize crontabs.
func (c *Controller) ScanAndCrontab() error {
	ds, err := c.cs.AppsV1().Deployments(CurNs).List(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("cannot list deployments: %w", err)
	}

	crons2 := make(map[uint32]*cron.Cron)
	for _, deploy := range ds.Items {
		if cronConfig, ok := deploy.Annotations[CronHpaConfigKey]; ok {
			hash := Hash(deploy.Name, cronConfig)

			if cr, ok := c.crons[hash]; ok {
				// If such deployment and config already exists, just simple delete from the
				// old map and adds to new one.
				crons2[hash] = cr
				delete(c.crons, hash)
			} else {
				// If not exists, which means the config is changed or deployment is new,
				// create a new cron object and adds to the new map.
				logrus.Infof("Job will be added: %s@\"%s\"", deploy.Name, cronConfig)
				cr = cron.New()
				c.addCronJobs(cr, cronConfig, deploy)
				crons2[hash] = cr
			}
		}
	}

	// The rest crons in the old map should be stopped and deleted.
	if len(c.crons) > 0 {
		logrus.Infof("%v outdated job(s) will be deleted.", len(c.crons))
	}
	for _, cr := range c.crons {
		cr.Stop()
	}

	c.crons = crons2
	return nil
}

// scalingJob Job of scaling.
func (c *Controller) scalingJob(replicas, minReplicas, maxReplicas uint32, deploy v12.Deployment) func() {
	return func() {
		// Fetch the newest version of deployment
		newDeploy, err := c.cs.AppsV1().Deployments(CurNs).Get(deploy.Name, v1.GetOptions{})
		if err != nil {
			logrus.Errorf("Cannot get deployment: %v", err.Error())
		}
		if replicas != 0 || (minReplicas == 0 && maxReplicas == 0) {
			*newDeploy.Spec.Replicas = int32(replicas)
			_, err := c.cs.AppsV1().Deployments(CurNs).Update(newDeploy)
			if err != nil {
				logrus.Errorf("Cannot update replicas of \"%s\": %s", newDeploy.Name, err.Error())
			} else {
				logrus.Infof("\"%s\" replicas is set to %v", newDeploy.Name, replicas)
			}
		}

		hpa := FindCorrespondingHpa(deploy, c.hpas)
		// Fetch the newest version of hpa
		if hpa != nil {
			hpa, _ = c.cs.AutoscalingV2beta2().HorizontalPodAutoscalers(CurNs).Get(hpa.Name, v1.GetOptions{})
		}

		if minReplicas != 0 || maxReplicas != 0 {
			if hpa == nil {
				return
			}
			if minReplicas != 0 {
				*hpa.Spec.MinReplicas = int32(minReplicas)
			}
			if maxReplicas != 0 {
				hpa.Spec.MaxReplicas = int32(maxReplicas)
			}
			hpa, err := c.cs.AutoscalingV2beta2().HorizontalPodAutoscalers(CurNs).Update(hpa)
			if err != nil {
				logrus.Errorf("Cannot update hpa object \"%s\": %s", hpa.Name, err.Error())
			} else {
				logrus.Infof("\"%s\" is set to: minReplicas: %v, maxReplicas: %v", hpa.Name,
					*hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas)
			}
		}
	}
}

// addCronJobs Adds cron jobs in form of JSON in `conf` to `cr`.
func (c *Controller) addCronJobs(cr *cron.Cron, conf string, deploy v12.Deployment) {
	var cronCfgs[] struct {
		Schedule string `json:"schedule"`
		Replicas uint32 `json:"replicas,omitempty"`
		MinReplicas uint32 `json:"minReplicas,omitempty"`
		MaxReplicas uint32 `json:"maxReplicas,omitempty"`
	}

	err := json.Unmarshal([]byte(conf), &cronCfgs)
	if err != nil {
		logrus.Errorf("Error parse cron scaling jobs: %v", err.Error())
		return
	}

	for _, cronCfg := range cronCfgs {
		err = cr.AddFunc(cronCfg.Schedule, c.scalingJob(cronCfg.Replicas, cronCfg.MinReplicas, cronCfg.MaxReplicas,
			deploy))
		logrus.Debugf("Job added: %s | %v | %v | %v", deploy.Name,
			cronCfg.Replicas, cronCfg.MinReplicas, cronCfg.MaxReplicas)
		if err != nil {
			logrus.Errorf("Cannot add cron job: %v", err.Error())
			continue
		}
	}
	cr.Start()
	logrus.Infof("Cron jobs for \"%s\" is started.", deploy.Name)
}