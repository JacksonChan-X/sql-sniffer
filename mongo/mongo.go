package mongo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/40t/go-sniffer/plugSrc/mongodb/build/bson"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/sirupsen/logrus"
)

type MongoDBStreamFactory struct {
	Logger *logrus.Logger
	Port   string
}

type MongoDBStream struct {
	net, transport gopacket.Flow
	buf            tcpreader.ReaderStream
}

type MongoDB struct {
	port   string
	source map[string]*stream
	mutex  sync.Mutex
	logger *logrus.Logger
}

type stream struct {
	packets   chan *packet
	logger    *logrus.Logger
	privateIp string
	publicIp  string
}

type packet struct {
	isClientFlow bool
	length       int
	requestID    uint32
	responseTo   uint32
	opCode       int // request type
	payload      io.Reader
}

// OpMsg 代表 OP_MSG 结构
type OpMsg struct {
	FlagBits  uint32
	Sections  []bson.M
	Checksum  uint32 // 可选的 CRC-32C 校验和
	HasCRC32C bool   // 是否包含 CRC-32C
}

type Section struct {
	PayloadType byte
	Document    bson.M
	Identifier  string
	Documents   []bson.M
}

var MongoDBInstance *MongoDB
var ErrTimeOut = errors.New("stream timeout")

func NewInstance(port string, logger *logrus.Logger) *MongoDB {
	if MongoDBInstance == nil {
		MongoDBInstance = &MongoDB{
			port:   port,
			source: make(map[string]*stream),
			logger: logger,
			mutex:  sync.Mutex{},
		}
	}
	return MongoDBInstance
}

func (m *MongoDBStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	ps := &MongoDBStream{
		net:       net,
		transport: transport,
		buf:       tcpreader.NewReaderStream(),
	}

	mongodbInstance := NewInstance(m.Port, m.Logger)
	go mongodbInstance.ResolveStream(net, transport, &ps.buf)

	return &ps.buf
}

func (m *MongoDB) ResolveStream(net, transport gopacket.Flow, buf io.Reader) {
	streamID := fmt.Sprintf("%v:%v", net.FastHash(), transport.FastHash())

	if _, ok := m.source[streamID]; !ok {
		stream := &stream{
			packets: make(chan *packet, 100),
			logger:  m.logger,
		}

		if transport.Dst().String() == m.port {
			stream.publicIp = net.Src().String()
			stream.privateIp = net.Dst().String()
		} else {
			stream.publicIp = net.Dst().String()
			stream.privateIp = net.Src().String()
		}

		m.mutex.Lock()
		m.source[streamID] = stream
		m.mutex.Unlock()
		go stream.run()
	}

	for {
		newPacket := m.newPacket(net, transport, buf)
		if newPacket == nil {
			return
		}
		m.source[streamID].packets <- newPacket
	}
}

func (m *MongoDB) newPacket(net, transport gopacket.Flow, buf io.Reader) *packet {
	var packet *packet
	var err error
	packet, err = readStream(buf)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		m.logger.Error(fmt.Sprintf("ERR:Unknown Stream: %s", err))
		return nil
	}

	if transport.Dst().String() == m.port {
		packet.isClientFlow = true
	} else {
		packet.isClientFlow = false
	}

	return packet
}

func readStream(r io.Reader) (*packet, error) {

	var buf bytes.Buffer
	p := &packet{}

	//header
	header := make([]byte, 16)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// message length
	p.length = int(binary.LittleEndian.Uint32(header[0:4]) - 16)
	p.requestID = binary.LittleEndian.Uint32(header[4:8])
	p.responseTo = binary.LittleEndian.Uint32(header[8:12])
	p.opCode = int(binary.LittleEndian.Uint32(header[12:]))

	if p.length != 0 {
		n, err := io.CopyN(&buf, r, int64(p.length))
		if err != nil {
			return nil, err
		}
		if n != int64(p.length) {
			return nil, fmt.Errorf("read payload length, incomplete")
		}
	}
	p.payload = bytes.NewReader(buf.Bytes())
	return p, nil
}

func (stm *stream) run() error {
	for {
		select {
		case packet, ok := <-stm.packets:
			if !ok {
				return nil
			}
			if packet.isClientFlow {
				stm.resolveClientPacket(packet)
			} else {
				// stm.resolveServerPacket(packet)
			}
		case <-time.After(time.Minute * 5): // 5分钟没有数据包，则认为连接断开
			return ErrTimeOut
		}
	}
}

func (stm *stream) resolveClientPacket(packet *packet) {
	var msg string
	switch packet.opCode {
	case OP_UPDATE:
		zero := ReadInt32(packet.payload)
		fullCollectionName := ReadString(packet.payload)
		flags := ReadInt32(packet.payload)
		selector := ReadBson2Json(packet.payload)
		update := ReadBson2Json(packet.payload)
		_ = zero
		_ = flags

		msg = fmt.Sprintf(" [OP_UPDATE] [coll:%s] %v %v",
			fullCollectionName,
			selector,
			update,
		)

	case OP_INSERT:
		flags := ReadInt32(packet.payload)
		fullCollectionName := ReadString(packet.payload)
		command := ReadBson2Json(packet.payload)
		_ = flags

		msg = fmt.Sprintf(" [OP_INSERT] [coll:%s] %v",
			fullCollectionName,
			command,
		)

	case OP_QUERY:
		flags := ReadInt32(packet.payload)
		fullCollectionName := ReadString(packet.payload)
		numberToSkip := ReadInt32(packet.payload)
		numberToReturn := ReadInt32(packet.payload)
		_ = flags
		_ = numberToSkip
		_ = numberToReturn

		command := ReadBson2Json(packet.payload)
		selector := ReadBson2Json(packet.payload)

		msg = fmt.Sprintf(" [OP_QUERY] [coll:%s] %v %v",
			fullCollectionName,
			command,
			selector,
		)

		// 如果selector是isMaster命令
		if strings.Contains(msg, "isMaster") {
			msg = ""
		}

	case OP_COMMAND:
		database := ReadString(packet.payload)
		commandName := ReadString(packet.payload)
		metaData := ReadBson2Json(packet.payload)
		commandArgs := ReadBson2Json(packet.payload)
		inputDocs := ReadBson2Json(packet.payload)

		msg = fmt.Sprintf(" [OP_COMMAND] [DB:%s] [Cmd:%s] %v %v %v",
			database,
			commandName,
			metaData,
			commandArgs,
			inputDocs,
		)

	case OP_GET_MORE:
		zero := ReadInt32(packet.payload)
		fullCollectionName := ReadString(packet.payload)
		numberToReturn := ReadInt32(packet.payload)
		cursorId := ReadInt64(packet.payload)
		_ = zero

		msg = fmt.Sprintf(" [OP_GET_MORE] [coll:%s] [num of reply:%v] [cursor:%v]",
			fullCollectionName,
			numberToReturn,
			cursorId,
		)

	case OP_DELETE:
		zero := ReadInt32(packet.payload)
		fullCollectionName := ReadString(packet.payload)
		flags := ReadInt32(packet.payload)
		selector := ReadBson2Json(packet.payload)
		_ = zero
		_ = flags

		msg = fmt.Sprintf(" [OP_DELETE] [coll:%s] %v",
			fullCollectionName,
			selector,
		)

	case OP_MSG:
		// stm.logger.Warn(fmt.Sprintf("OP_MSG: %+v", packet))
		payload, err := io.ReadAll(packet.payload)
		if err != nil {
			stm.logger.Error(fmt.Sprintf("read payload error: %v", err))
			return
		}
		_, msg = parseSections(payload)
	default:
		return
	}

	if len(msg) == 0 {
		return
	}
	stm.logger.Info(fmt.Sprintf("%s->%s:%s", stm.publicIp, stm.privateIp, msg))
}

// func (stm *stream) resolveServerPacket(packet *packet) {
// 	return
// }
