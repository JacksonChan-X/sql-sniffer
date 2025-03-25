package mysql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/JacksonChan-X/sql-sniffer/client"
	"github.com/JacksonChan-X/sql-sniffer/helper"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/sirupsen/logrus"
)

type MysqlStreamFactory struct {
	Logger *logrus.Logger
	Port   string
}

type MysqlStream struct {
	net, transport gopacket.Flow
	buf            tcpreader.ReaderStream
}

type Mysql struct {
	Port      string
	StreamMap map[string]*Stream
	logger    *logrus.Logger
	mutex     sync.Mutex
}

type Stream struct {
	ID        string
	Packet    chan *Packet
	StmtMap   map[uint32]*Statement
	Seq       chan *Packet
	needSeq   chan bool
	logger    *logrus.Logger
	privateIP string
	publicIP  string
}

type Packet struct {
	IsClientFlow bool
	Seq          uint8
	Length       int
	Payload      []byte
}

var (
	mysql *Mysql
	once  sync.Once
)

func NewInstance(port string, logger *logrus.Logger) *Mysql {
	once.Do(func() {
		mysql = &Mysql{
			Port:      port,
			StreamMap: make(map[string]*Stream),
			logger:    logger,
			mutex:     sync.Mutex{},
		}
	})
	return mysql
}

func (p *MysqlStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	ps := &MysqlStream{
		net:       net,
		transport: transport,
		buf:       tcpreader.NewReaderStream(),
	}

	mysqlInstance := NewInstance(p.Port, p.Logger)
	go mysqlInstance.ResolveStream(net, transport, &ps.buf)

	return &ps.buf
}

func (m *Mysql) ResolveStream(net, transport gopacket.Flow, buf io.Reader) {
	streamID := fmt.Sprintf("%v:%v", net.FastHash(), transport.FastHash())

	m.mutex.Lock()
	if _, ok := m.StreamMap[streamID]; !ok {
		stream := &Stream{
			ID:      streamID,
			Packet:  make(chan *Packet, 100),
			StmtMap: make(map[uint32]*Statement, 0),
			logger:  m.logger,
			Seq:     make(chan *Packet, 100),
			needSeq: make(chan bool, 100),
		}
		if transport.Dst().String() == m.Port {
			stream.privateIP = net.Dst().String()
			stream.publicIP = net.Src().String()
		} else {
			stream.privateIP = net.Src().String()
			stream.publicIP = net.Dst().String()
		}
		m.StreamMap[streamID] = stream
		m.mutex.Unlock()
		go stream.run()
	} else {
		m.mutex.Unlock()
	}

	for {
		// 解析新包
		newPacket := m.newPacket(net, transport, buf)
		if newPacket == nil {
			if transport.Src().String() == m.Port {
				m.mutex.Lock()
				close(m.StreamMap[streamID].Seq)
				m.mutex.Unlock()
			} else {
				m.mutex.Lock()
				close(m.StreamMap[streamID].Packet)
				m.mutex.Unlock()
			}
			return
		}

		if newPacket.IsClientFlow {
			m.mutex.Lock()
			if cmd := newPacket.Payload[0]; cmd == COM_STMT_PREPARE {
				m.StreamMap[streamID].needSeq <- true
			}
			m.StreamMap[streamID].Packet <- newPacket // 客户端包
			m.mutex.Unlock()
		} else {
			if newPacket.Seq != 1 {
				continue
			}
			// m.logger.Warn("seq == 1")
			m.mutex.Lock()
			select {
			case need, ok := <-m.StreamMap[streamID].needSeq:
				// m.logger.Warn("receive")
				if !ok {
					continue
				}
				if need {
					m.StreamMap[streamID].Seq <- newPacket
					// m.logger.Warn(fmt.Sprintf("%+v", newPacket))
				}
			default:
			}
			m.mutex.Unlock()
		}
	}
}

func (m *Mysql) newPacket(net, transport gopacket.Flow, r io.Reader) *Packet {

	//read packet
	var payload *bytes.Buffer
	var seq uint8
	var err error
	if seq, payload, err = m.resolvePacket(r); err != nil {
		if err == io.EOF {
			m.logger.Info(fmt.Sprintf("stream:%s close",
				net.Src().String()+":"+transport.Src().String()+":"+
					net.Dst().String()+":"+transport.Dst().String(),
			))
			return nil
		} else {
			m.logger.Error(fmt.Printf("ERR : Unknown Packet, stream:%s,err:%s",
				net.Src().String()+":"+transport.Src().String()+":"+net.Dst().String()+":"+transport.Dst().String(),
				err,
			))
		}
	}

	//generate new packet
	var pk = Packet{
		Seq:     seq,
		Length:  payload.Len(),
		Payload: payload.Bytes(),
	}
	if transport.Src().String() == m.Port {
		pk.IsClientFlow = false
	} else {
		pk.IsClientFlow = true
	}

	return &pk
}

func (m *Mysql) resolvePacket(r io.Reader) (uint8, *bytes.Buffer, error) {
	header := make([]byte, 4)
	if n, err := io.ReadFull(r, header); err != nil {
		if n == 0 && err == io.EOF {
			return 0, nil, io.EOF
		}
		return 0, nil, ErrorStream
	}
	length := getUint24(header[0:3])
	seq := header[3]
	payload := new(bytes.Buffer)
	n, err := io.CopyN(payload, r, int64(length))
	if err != nil {
		return 0, nil, err
	}
	if n != int64(length) {
		return 0, nil, ErrorStream
	}
	return seq, payload, nil
}

func (stm *Stream) run() error {
	for {
		select {
		case Packet, ok := <-stm.Packet:
			if !ok {
				return nil
			}
			if Packet.Length != 0 {
				if Packet.IsClientFlow {
					stm.resolveClientPacket(Packet)
				} else {
					// go stm.resolveServerPacket(Packet)
				}
			}
		case <-time.After(time.Minute * 5): // 5分钟没有数据包，则认为连接断开
			return ErrTimeOut
		}
	}
}

func (stm *Stream) findStmtPacket(seq uint8) *Packet {
	for {
		select {
		case packet, ok := <-stm.Seq:
			if !ok {
				return nil
			}
			if !packet.IsClientFlow && packet.Seq == seq {
				return packet
			} else {
				continue
			}
		case <-time.After(10 * time.Second):
			stm.logger.Warn("timeout")
			return nil
		}
	}
}

func (stm *Stream) resolveClientPacket(p *Packet) {
	payload := p.Payload
	seq := p.Seq

	var msg = ""
	if len(payload) == 0 {
		return
	}

	cmd := payload[0]
	data := payload[1:]
	switch cmd {
	case COM_INIT_DB:
		msg = fmt.Sprintf("USE %s;\n", data)
	case COM_DROP_DB:
		msg = fmt.Sprintf("Drop DB %s;\n", data)
	case COM_CREATE_DB, COM_QUERY:
		msg = string(data)
	case COM_STMT_PREPARE:
		serverPacket := stm.findStmtPacket(seq + 1)
		if serverPacket == nil {
			stmt := &Statement{
				SQL:        string(data),
				ParamCount: helper.GetParamCount(string(data)),
			}
			stmt.Args = make([]any, stmt.ParamCount)
			stm.StmtMap[0] = stmt
			stm.logger.Error(fmt.Sprintf("ERR : Not found seq:%d,sql:%s", seq+1, string(data)))
			return
		}

		stmtID := binary.LittleEndian.Uint32(serverPacket.Payload[1:5])
		stmt := &Statement{
			ID:         stmtID,
			SQL:        string(data),
			FieldCount: binary.LittleEndian.Uint16(serverPacket.Payload[5:7]),
			ParamCount: binary.LittleEndian.Uint16(serverPacket.Payload[7:9]),
		}
		stmt.Args = make([]any, stmt.ParamCount)
		stm.StmtMap[stmtID] = stmt
	case COM_STMT_EXECUTE:
		var (
			ok       bool
			stmt, ts *Statement
			pos      = 1
		)
		stmtID := binary.LittleEndian.Uint32(payload[pos : pos+4])
		stmt, ok = stm.StmtMap[stmtID]
		if !ok {
			if ts, ok = stm.StmtMap[0]; ok {
				stmt = ts
			} else {
				stm.logger.Error(fmt.Sprintf("ERR : Not found stmtID:%d", stmtID))
				return
			}
		}
		pos = 5 // pos = 5

		var nullBitmaps, paramTypes, paramValues []byte
		pos += 4
		if stmt.ParamCount > 0 {
			nullBitmapLen := (stmt.ParamCount + 7) >> 3
			if len(data) < (pos + int(nullBitmapLen) + 1) {
				stm.logger.Warn("ERR:Malform packet error")
			}
			nullBitmaps = data[pos : pos+int(nullBitmapLen)]
			pos += int(nullBitmapLen)

			// new param bound flag
			if data[pos] == 1 {
				pos++
				if len(data) < (pos + int(stmt.ParamCount<<1)) {
					stm.logger.Warn("ERR:Malform packet error")
				}

				paramTypes = data[pos : pos+int(stmt.ParamCount<<1)]
				pos += int(stmt.ParamCount << 1)

				paramValues = data[pos:]

				if err := stmt.BindStmtArgs(nullBitmaps, paramTypes, paramValues); err != nil {
					stm.logger.Error(fmt.Sprintf("ERR : Could not bind params,%s", err.Error()))
				}
			}
			msg = client.ExplainSQL(stmt.SQL, nil, `'`, stmt.Args...)
		}
	case COM_QUIT:
		msg = fmt.Sprintf("QUIT stream:%s", stm.ID)
	case COM_STMT_CLOSE:
		stmtID := binary.LittleEndian.Uint32(payload[1:5])
		msg = fmt.Sprintf("Close,stream:%s,stmtID:%d", stm.ID, stmtID)
	default:
		return
	}

	stm.logger.Info(stm.publicIP + ":" + stm.privateIP + " " + msg)
}

// func (stm *Stream) resolveServerPacket(p *Packet) {
// 	if len(p.Payload) == 0 {
// 		return
// 	}

// 	cmd := p.Payload[0]
// 	switch cmd {
// 	case 0x00:
// 		stmtID := binary.LittleEndian.Uint32(p.Payload[1:5])
// 		if stmtID == stm.curStmtID {
// 			stm.logger.Info(fmt.Sprintf("success stmtID:%d,stream:%s,packet:%+v", stmtID, stm.ID, p))
// 			stm.curStmtID += 1
// 			stm.Seq <- p
// 		} else {
// 			stm.logger.Warn(fmt.Sprintf("ERR : Not found stmtID:%d,stream:%s,packet:%+v", stmtID, stm.ID, p))
// 		}
// 	default:
// 		return
// 	}
// }
