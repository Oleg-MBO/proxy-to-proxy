package main

import (
	"fmt"
	"os"

	"github.com/oov/socks5"
	log "github.com/sirupsen/logrus"

	"github.com/Oleg-MBO/proxy-to-proxy/socks5Server"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	// log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func main() {
	// proxyConf := ProxySocks5Conf{
	// 	Address: "188.120.234.190:9261",
	// }
	// lat, err := proxyConf.CheckLatency()
	// checkErr(err)
	// fmt.Printf("latensy %f s\n", lat)
	// fmt.Printf("out ip is %s\n", string(proxyConf.OutIP))

	// fmt.Printf("country is %s\n", proxyConf.CountryIsoCode)

	proxyList, err := GetProxiesList("RU")
	// for _, proxy := range proxyList {
	// 	latency, err := proxy.CheckLatency()
	// 	if err != nil {
	// 		log.Println(err)
	// 		continue
	// 	}
	// 	// log.Printf("%s latency is %.5f s", proxy.Address, latency)
	// }
	checkErr(err)

	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	roundRobProxy := NewProxyRoundRobin(logger)
	roundRobProxy.AddProxyConfigs(proxyList...)

	dialFunc := roundRobProxy.GetDialFunc()

	srv := socks5Server.New(dialFunc)
	srv.AuthUsernamePasswordCallback = func(c *socks5Server.Conn, username, password []byte) error {
		user := string(username)
		if user != "guest" {
			return socks5.ErrAuthenticationFailed
		}

		log.Printf("Welcome %v!", user)
		c.Data = user
		return nil
	}
	srv.HandleConnectFunc(func(c *socks5Server.Conn, host string) (newHost string, err error) {
		if host == "example.com:80" {
			return host, socks5.ErrConnectionNotAllowedByRuleset
		}
		if user, ok := c.Data.(string); ok {
			log.Printf("%v connecting to %v", user, host)
		}
		return host, nil
	})
	srv.HandleCloseFunc(func(c *socks5Server.Conn) {
		if user, ok := c.Data.(string); ok {
			log.Printf("Goodbye %v!", user)
		}
	})

	srv.ListenAndServe(":12345")

}
