# A simple build script, just for specify version info
CGO_ENABLED=0 GOOS=linux  go build -o bin/jsonbox_exporter \
        -ldflags "-X github.com/prometheus/common/version.Version=`cat VERSION` -X github.com/prometheus/common/version.Revision=`git rev-parse HEAD` -X github.com/prometheus/common/version.Branch=`git rev-parse --abbrev-ref HEAD` -X github.com/prometheus/common/version.BuildUser=`whoami` -X github.com/prometheus/common/version.BuildDate=`date +%Y%m%d-%H:%M:%S`"  \
        -v -a -trimpath jsonbox_exporter/cmd
