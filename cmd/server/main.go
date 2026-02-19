package main

import (
	"flag"

	"github.com/joeblew999/plat-mjml/internal/config"
	"github.com/joeblew999/plat-mjml/internal/server"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	configFile := flag.String("f", "etc/plat-mjml.yaml", "config file path")
	flag.Parse()

	logx.DisableStat()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	s, err := server.New(c)
	logx.Must(err)

	s.Start()
}
