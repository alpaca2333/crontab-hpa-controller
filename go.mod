module cron-hpa-controller

require (
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/testify v1.2.2
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v12.0.0+incompatible
)

go 1.13
