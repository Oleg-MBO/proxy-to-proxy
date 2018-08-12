package main

import (
	"fmt"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

type ProxyRoundRobin struct {
	proxyConfigs []ProxySocks5Conf

	n     int
	mutex sync.Mutex

	log *log.Logger
}

func NewProxyRoundRobin(logg *log.Logger) *ProxyRoundRobin {
	pr := new(ProxyRoundRobin)
	pr.log = logg
	return pr
}

// AddProxyConf check and register proxy for use
func (prr *ProxyRoundRobin) AddProxyConfigs(prConfs ...ProxySocks5Conf) {
	go func(prConfs []ProxySocks5Conf) {
		for _, PrConf := range prConfs {
			if PrConf.Address != "" && !prr.isProxyExist(PrConf) {
				latency, err := PrConf.CheckLatency()
				if err != nil {
					prr.log.WithError(err).WithField("addres", PrConf.Address).Debug("can`t add proxy to list")
					continue
				}
				if latency > 10 {
					prr.log.WithField("addres", PrConf.Address).WithField("latency", latency).Debug("can`t add proxy to list, latency>10")
					continue
				}
				prr.addProxyConfToList(PrConf)
			}
		}
	}(prConfs)
}

func (prr *ProxyRoundRobin) addProxyConfToList(prConf ProxySocks5Conf) {
	if prr.isProxyExist(prConf) {
		return
	}
	prr.mutex.Lock()
	defer prr.mutex.Unlock()
	prr.log.WithField("addres", prConf.Address).WithField("latency", prConf.Latency).Debug("add proxy to list")
	prr.proxyConfigs = append(prr.proxyConfigs, prConf)
}
func (prr *ProxyRoundRobin) rmProxyConfFromList(prConf ProxySocks5Conf) {
	prr.mutex.Lock()
	defer prr.mutex.Unlock()
	for i, myPrConf := range prr.proxyConfigs {
		if myPrConf.Address == prConf.Address {
			prr.proxyConfigs = append(prr.proxyConfigs[:i], prr.proxyConfigs[i+1:]...)
		}
	}
}

func (prr *ProxyRoundRobin) isProxyExist(prConf ProxySocks5Conf) bool {
	prr.mutex.Lock()
	defer prr.mutex.Unlock()
	for _, myPrConf := range prr.proxyConfigs {
		if myPrConf.Address == prConf.Address {
			return true
		}
	}
	return false
}

func (prr *ProxyRoundRobin) GetDialFunc() func(network, address string) (net.Conn, error) {
	return func(network, address string) (net.Conn, error) {
		prr.mutex.Lock()
		// lenProxies := len(prr.proxyConfigs)
		if len(prr.proxyConfigs) == 0 {
			return nil, fmt.Errorf("does not have availible proxies")
		}
		if prr.n >= len(prr.proxyConfigs) {
			prr.n = 0
		}
		conf := prr.proxyConfigs[prr.n]
		prr.log.WithField("proxy", prr.n).WithField("adress", conf.Address).Info("using proxy")
		prr.n++
		prr.mutex.Unlock()

		dialer, err := conf.GetDialer()
		if err != nil {
			return nil, err
		}

		return dialer.Dial(network, address)
	}
}

func (prr *ProxyRoundRobin) CheckProxiesWork() {
	prr.log.Info("start checking proxies work")
	defer func() {
		prr.mutex.Lock()
		prr.log.WithField("active_proxies", len(prr.proxyConfigs)).Info("finish check proxies work")
		prr.mutex.Unlock()
	}()

	checkedProxyAddrMap := make(map[string]bool)

	for {
		var prConf ProxySocks5Conf
		prr.mutex.Lock()

		for _, prConf = range prr.proxyConfigs {
			if !checkedProxyAddrMap[prConf.Address] {
				checkedProxyAddrMap[prConf.Address] = true
				break
			}
		}
		prr.mutex.Unlock()
		if prConf.Address == "" {
			// all proxies is checked
			return
		}
		latency, err := prConf.CheckLatency()
		if err != nil {
			prr.log.WithField("addres", prConf.Address).WithField("error", err).Debug("error during check proxy, removing it")
			prr.rmProxyConfFromList(prConf)
		}
		if latency >= 10 {
			prr.log.WithField("addres", prConf.Address).WithField("latency", latency).Debug("to long lantency during check proxy, removing it")
			prr.rmProxyConfFromList(prConf)
		}
	}
}
