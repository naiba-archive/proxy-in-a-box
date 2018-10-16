package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/naiba/proxyinabox/mitm"
	"github.com/naiba/proxyinabox/service"

	"github.com/naiba/com"

	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/naiba/proxyinabox"
	"github.com/naiba/proxyinabox/crawler"
)

var configFilePath, httpProxyAddr, httpsProxyAddr string
var m *mitm.MITM
var rootCmd = &cobra.Command{
	Use:   "proxy-in-a-box",
	Short: "Proxy-in-a-Box provide many proxies.",
	Long:  `Proxy-in-a-Box helps programmers quickly and easily develop powerful crawler services. one-script, easy-to-use: proxies in a box.`,
	Run: func(cmd *cobra.Command, args []string) {
		proxyinabox.Init()
		fmt.Println("[PIAB]", "main", "[😁]", proxyinabox.Config.Sys.Name, "v1.0.0")
		proxyinabox.CI = service.NewMemCache()

		crawler.Init()

		m = newMITM()
		m.Init()

		m.ServeHTTP()

		crawler.FetchProxies()
		crawler.Verify()

		c := cron.New()
		c.AddFunc("@daily", crawler.FetchProxies)
		c.AddFunc("0 "+strconv.Itoa(proxyinabox.Config.Sys.VerifyDuration)+" * * * *", crawler.Verify)
		c.Start()

		select {}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		//read config
		viper.SetConfigType("yaml")
		viper.SetConfigFile(configFilePath)
		com.PanicIfNotNil(viper.ReadInConfig())
		com.PanicIfNotNil(viper.Unmarshal(&proxyinabox.Config))
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFilePath, "conf", "c", "./pb.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&httpProxyAddr, "ha", "p", "127.0.0.1:8080", "http proxy server addr")
	rootCmd.PersistentFlags().StringVarP(&httpsProxyAddr, "sa", "s", "127.0.0.1:8081", "https proxy server addr")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("[PIAB]", "panic", "[👻]", err)
		os.Exit(1)
	}
}

func newMITM() *mitm.MITM {
	return &mitm.MITM{
		ListenHTTPS: true,
		HTTPAddr:    httpProxyAddr,
		HTTPSAddr:   httpsProxyAddr,
		TLSConf: &mitm.TLSConfig{
			PrivateKeyFile: "proxyinabox.key",
			CertFile:       "proxyinabox.pem",
		},
		Print:     proxyinabox.Config.Debug,
		IsDirect:  false,
		Scheduler: proxyinabox.CI.PickProxy,
		Filter: func(req *http.Request) error {
			if req.Header.Get("Naiba") != "lifelonglearning" {
				return fmt.Errorf("%s", "Proxy Authentication Required")
			}
			req.Header.Del("Naiba")
			if !proxyinabox.CI.IPLimiter(req) {
				return fmt.Errorf("%s", "请求次数过快")
			}
			if !proxyinabox.CI.HostLimiter(req) {
				return fmt.Errorf("%s", "请求域名过多")
			}
			return nil
		},
	}
}
