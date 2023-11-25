package config

import (
	"log/slog"
	"net"

	"github.com/spf13/viper"
)

var LocalIP string
var WebSocketPort int

func Init() {
	loadConfig()
	LocalIP = getLocalIP()
	WebSocketPort = GetInt("websocket.port")
	if LocalIP == "" || WebSocketPort == 0 {
		slog.Error("Can not find IP or Port")
	}
	slog.Info("Config Init", "LocalIP", LocalIP, "WebSocketPort", WebSocketPort)
}

func loadConfig() {
	slog.Info("Config Loaded")
	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read config file", "error", err)
		return
	}
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
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
