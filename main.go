package main

import (
	"fmt"
	"os"
	"time"

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

	// proxyList, err := GetProxiesList("RU")
	// checkErr(err)

	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	roundRobProxy := NewProxyRoundRobin(logger)

	updateProxiesTicker := time.Tick(time.Minute * 45)

	go func() {
		log.Info("getting proxies list")
		proxyList, err := GetProxiesList("RU")
		if err != nil {
			log.Warningf("could not GetProxiesList: %#v", err)
			roundRobProxy.AddProxyConfigs(proxyList...)
		}
		log.Info("start adding new proxies if exist")
		roundRobProxy.AddProxyConfigs(proxyList...)
		<-updateProxiesTicker
	}()

	checkProxiesTicker := time.Tick(time.Minute * 90)

	go func() {
		<-checkProxiesTicker
		roundRobProxy.CheckProxiesWork()
	}()

	dialFunc := roundRobProxy.GetDialFunc()

	srv := socks5Server.New(dialFunc)
	srv.AuthUsernamePasswordCallback = func(c *socks5Server.Conn, username, password []byte) error {
		user := string(username)
		if user == "" {
			user = "undefined user"
		}
		// if user != "guest" {
		// 	return socks5.ErrAuthenticationFailed
		// }

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

	servAdrr := "0.0.0.0:1234"
	log.WithField("addres", servAdrr).Info("start proxy sever")

	checkErr(srv.ListenAndServe("0.0.0.0:1234"))

}
