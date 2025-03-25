package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	interfaces, mysqlPorts, mongoPorts, redisPorts string
	logger                                         *logrus.Logger
	debug                                          bool
)

var rootCmd = &cobra.Command{
	Use:   "SQL-Sniffer -i [interface] -mysql_port [port] -mongo_port [port] -redis_port [port]",
	Short: "MySQL、MongoDB和Redis流量嗅探工具",
	Long: `mysql-sniffer是一个网络流量嗅探工具，
可以捕获并分析MySQL、MongoDB和Redis的网络流量。`,
	Example: "sql-sniffer -i eth0 -mysql_port 3306,3307 -mongo_port 27017",
	Run:     sniffer,
}

// Execute 添加所有子命令到根命令并设置标志。
// 这是由main.main()调用的。只需要对rootCmd调用一次。
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&interfaces, "interfaces", "i", "", "要监听的网络接口，逗号分隔")
	rootCmd.PersistentFlags().StringVar(&mysqlPorts, "mysql_port", "3306", "MySQL端口，逗号分隔")
	rootCmd.PersistentFlags().StringVar(&mongoPorts, "mongo_port", "27017", "MongoDB端口，逗号分隔")
	rootCmd.PersistentFlags().StringVar(&redisPorts, "redis_port", "6379", "Redis端口，逗号分隔")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "启用调试模式")
}
