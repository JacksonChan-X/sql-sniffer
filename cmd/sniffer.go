package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sql-sniffer/helper"
	"sql-sniffer/mongo"
	"sql-sniffer/mysql"
	"sql-sniffer/redis"
	"sql-sniffer/server"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const (
	maxPacketSize = 1600
)

const (
	MYSQL = "mysql"
	MONGO = "mongo"
	REDIS = "redis"
)

func sniffer(cmd *cobra.Command, args []string) {
	// 初始化日志
	logger = server.NewLogger(debug)

	mysqlPortList := strings.Split(mysqlPorts, ",")
	mongoPortList := strings.Split(mongoPorts, ",")
	redisPortList := strings.Split(redisPorts, ",")
	interList, err := helper.GetAllInterfaces()
	if err != nil {
		logger.Fatal(fmt.Sprintf("获取网卡失败: %v", err))
	}
	if len(interfaces) != 0 {
		interList = strings.Split(interfaces, ",")
	}

	ctx, resetSignal := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	eg, ctx := errgroup.WithContext(ctx)

	for _, inter := range interList {
		for _, mysqlPort := range mysqlPortList {
			if len(mysqlPort) == 0 {
				continue
			}
			i, p := inter, mysqlPort
			eg.Go(func() error {
				FetchPacket(ctx, i, p, MYSQL)
				return nil
			})
		}

		for _, mongoPort := range mongoPortList {
			if len(mongoPort) == 0 {
				continue
			}
			i, p := inter, mongoPort
			eg.Go(func() error {
				FetchPacket(ctx, i, p, MONGO)
				return nil
			})
		}

		for _, redisPort := range redisPortList {
			if len(redisPort) == 0 {
				continue
			}
			i, p := inter, redisPort
			eg.Go(func() error {
				FetchPacket(ctx, i, p, REDIS)
				return nil
			})
		}
	}

	eg.Wait()

	<-ctx.Done()
	resetSignal()
}

func FetchPacket(ctx context.Context, inter, port, typ string) {
	handle, err := pcap.OpenLive(inter, maxPacketSize, true, time.Second)
	if err != nil {
		logger.Fatal(err)
	}
	defer handle.Close()

	err = handle.SetBPFFilter("tcp port " + port)
	if err != nil {
		logger.Fatal(err)
	}

	err = helper.GetLocalIpByInterface(inter)
	if err != nil {
		logger.Error(err)
		return
	}

	var streamFactory tcpassembly.StreamFactory
	switch typ {
	case MYSQL:
		streamFactory = &mysql.MysqlStreamFactory{Logger: logger, Port: port}
	case MONGO:
		streamFactory = &mongo.MongoDBStreamFactory{Logger: logger, Port: port}
	case REDIS:
		streamFactory = &redis.RedisStreamFactory{Logger: logger, Port: port}
	}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	ticker := time.Tick(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			// logger.Info("context done")
			return
		case pkt := <-packetSource.Packets():
			if pkt.NetworkLayer() == nil || pkt.TransportLayer() == nil ||
				pkt.TransportLayer().LayerType() != layers.LayerTypeTCP {
				logger.Info("Unusable packet")
				continue
			}
			tcp := pkt.TransportLayer().(*layers.TCP)

			assembler.AssembleWithTimestamp(pkt.NetworkLayer().NetworkFlow(), tcp, pkt.Metadata().Timestamp)
		case <-ticker:
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		}
	}
}
