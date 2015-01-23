package main
import (
	"flag"
	"log"
	"internal"
	"service"
	"fmt"
	"os"
	"runtime"
)

func main() {
        flag.Parse()
        conf, err := internal.ParseConf()

        if err != nil {
                log.Fatal(err)
        }
        localIp,err := service.GetLocalIp()
        if err != nil {
                log.Fatal(err)
        }
        // 注册到LOOKUP服务
        service.RegisterMoaService(&service.MoaServiceRegReq{
                ServiceUri:fmt.Sprintf("/service/prism/%s", conf.Namespace),
                HostPort:fmt.Sprintf("%s:%d", localIp, conf.Port),
                // redis是udp memcache是tcp
                Protocol:"redis",
        })
	runtime.GOMAXPROCS(runtime.NumCPU())
        log.Fatal(internal.NewServer(conf, os.Stdout).Listen(nil, nil, nil))
}
