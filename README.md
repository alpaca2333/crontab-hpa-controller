# Cron Hpa Controller
Crontab hpa scheduler. There is already a work [here](https://github.com/amelbakry/kube-schedule-scaler/). 
I found this repo's way of configuration is elegant but there is some issues about crontabbing inside a docker.
It is implemented in python so i decide to reimplement it with client-go.

## Usage
Install helm charts in the namespaces that contains deployments need to be scheduled.

```shell script
helm install cronhpacontroller chart/cronhpa [-n namespace]
```

And add an annotation named `qsun.tencent.com/cronhpa` in form of json.
You can set `replicas` (for those deployments who don't have an hpa) or `minReplicas` and `maxReplicas` 
(for hpas whose targets are the deployments with the annotation.)

```yaml
metadata:
  annotations:
    qsun.tencent.com/cronhpa: |
      [
        {"schedule": "1 * * * *", "replicas": 4, "replicas": 10},
        {"schedule": "1 * * * *", "replicas": 4, "minReplicas": 1, "maxReplicas": 20}
      ]
```

You can refer to the logs to check if the controller works as expected. Grep deployment's name from the log.
```
$> kubectl logs -f cronhpacontroller-7c5446789f-97wvk cronhpa | grep "act10"

  time="2020-04-03T02:24:55Z" level=info msg="Job will be added: act10@\"[\n  {\"schedule\": \"1 * * * *\", \"replicas\": 4, \"minReplicas\": 5},  \n  {\"schedule\": \"1 * * * *\", \"replicas\": 4, \"maxReplicas\": 6}\n]\n\""
  time="2020-04-03T02:24:55Z" level=debug msg="Job added: act10 | 4 | 5 | 0"
  time="2020-04-03T02:24:55Z" level=debug msg="Job added: act10 | 4 | 0 | 6"
  time="2020-04-03T02:24:55Z" level=info msg="Cron jobs for \"act10\" is started."
  time="2020-04-03T02:25:01Z" level=info msg="\"act10\" replicas is set to 4"
  time="2020-04-03T02:25:01Z" level=info msg="\"act10\" replicas is set to 4"
```

## Build
```shell script
# Requires go v1.13+
GOOS=linux go build -o ./bin/cronhpa -mod vendor ./cmd/cron-hpa-controller/main.go
# You can directly use csighub.tencentyun.com/qwertysun/cronhpa:v14 for that
Docker build -f build/Dockerfile -t cronhpa .
```

## TODO
+ Better log viewing experience.
+ Install a unique controller in the namespace `kube-system` and enable the function by labeling a namespace.
+ Docker multi-stages build.