package config

import (
	"log"
	"net"

	"github.com/spf13/viper"
)

var LocalIP string

func init() {
	loadConfig()
	LocalIP = getLocalIP()
	if LocalIP == "" {
		log.Fatalln("Can not find IP")
	}
	log.Println("LocalIP =", LocalIP)
}

func loadConfig() {
	log.Println("Config Loaded")
	viper.SetConfigFile("./config/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Println("Failed to read config file:", err)
		return
	}
}

func GetString(key string) string {
	return viper.GetString(key)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	var ip string
	for _, addr := range addrs {
		// IPv4 주소만 고려
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ip = ipnet.IP.String()
			break
		}
	}
	if ip == "" {
		return ""
	}
	return ip
}
