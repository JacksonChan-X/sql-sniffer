package redis

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/sirupsen/logrus"
)

type RedisStreamFactory struct {
	Logger *logrus.Logger
	Port   string
}

type RedisStream struct {
	net, transport gopacket.Flow
	buf            tcpreader.ReaderStream
}

type Redis struct {
	port   string
	logger *logrus.Logger
}

var (
	redis *Redis
	once  sync.Once
)

func NewInstance(port string, logger *logrus.Logger) *Redis {
	once.Do(func() {
		redis = &Redis{
			port:   port,
			logger: logger,
		}
	})
	return redis
}

func (p *RedisStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	ps := &RedisStream{
		net:       net,
		transport: transport,
		buf:       tcpreader.NewReaderStream(),
	}

	redisInstance := NewInstance(p.Port, p.Logger)
	go redisInstance.ResolveStream(net, transport, &ps.buf)

	return &ps.buf
}

func (m *Redis) ResolveStream(net, transport gopacket.Flow, r io.Reader) {
	buf := bufio.NewReader(r)
	var cmd string
	var cmdCount = 0
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			if err == io.EOF {
				m.logger.Info("redis stream end")
				return
			}
			m.logger.Error(fmt.Sprintf("redis stream read line error: %v", err))
			continue
		}

		if len(line) == 0 || transport.Src().String() == m.port {
			continue
		}

		// m.logger.Debug(fmt.Sprintf("Redis raw line: %s, src: %s, port: %s", string(line), transport.Src().String(), m.port))

		// 处理RESP协议
		if strings.HasPrefix(string(line), "*") {
			count, err := strconv.Atoi(string(line[1:]))
			if err != nil {
				m.logger.Error(fmt.Sprintf("parse command count error: %v, line: %s", err, string(line)))
				continue
			}

			cmdCount = count
			cmd = ""
			cmdParts := make([]string, 0, cmdCount)

			for j := 0; j < cmdCount; j++ {
				// 读取长度行 ($n)
				lenLine, _, err := buf.ReadLine()
				if err != nil {
					m.logger.Error(fmt.Sprintf("read length line error: %v", err))
					break
				}

				if !strings.HasPrefix(string(lenLine), "$") {
					m.logger.Error(fmt.Sprintf("invalid length line: %s", string(lenLine)))
					break
				}

				// 读取实际数据
				dataLine, _, err := buf.ReadLine()
				if err != nil {
					m.logger.Error(fmt.Sprintf("read data line error: %v", err))
					break
				}

				cmdParts = append(cmdParts, string(dataLine))
			}

			if len(cmdParts) > 0 {
				cmd = strings.Join(cmdParts, " ")
				m.logger.Info(fmt.Sprintf("Command: %s", cmd))
			}
		}
	}
}
