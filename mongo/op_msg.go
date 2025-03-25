package mongo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// ParseOpMsg 解析 OP_MSG (2013) 消息
// parseSections 解析 sections 字段
func parseSections(payload []byte) (string, string) {
	reader := bytes.NewReader(payload)
	var result strings.Builder

	// 读取 flagBits（4 字节）
	var flagBits uint32
	binary.Read(reader, binary.LittleEndian, &flagBits)
	// result.WriteString(fmt.Sprintf("Flag Bits: 0x%X\n", flagBits))

	// 检查是否包含校验和
	hasChecksum := (flagBits & 0x01) != 0
	// result.WriteString(fmt.Sprintf("Has Checksum: %v\n", hasChecksum))

	// 计算需要解析的数据长度（如果有校验和，则排除最后4字节）
	dataLength := reader.Len()
	if hasChecksum {
		dataLength -= 4
	}

	// 解析 sections
	sectionsStart := reader.Len()
	var commandName string
	var commandValue interface{}
	var database string
	var collection string

	for reader.Len() > (sectionsStart - dataLength) {
		sectionType, _ := reader.ReadByte()
		switch sectionType {
		case 0x00:
			// Type 0: 单个 BSON 文档
			// result.WriteString("Section Type 0 (Single BSON Document)\n")
			bsonDoc, err := readBSON(reader)
			if err != nil {
				result.WriteString(fmt.Sprintf("Error parsing BSON section: %v\n", err))
				return "", result.String()
			}

			// 尝试识别命令类型
			for k, v := range bsonDoc {
				// 跳过特殊字段
				if k == "$db" {
					if dbName, ok := v.(string); ok {
						database = dbName
					}
					continue
				}
				if k == "lsid" || k == "txnNumber" || k == "$clusterTime" {
					continue
				}

				// 第一个非特殊字段通常是命令名称
				if commandName == "" {
					commandName = k
					commandValue = v

					// 如果是集合名称，保存它
					if strVal, ok := v.(string); ok {
						collection = strVal
					}
				}
			}

			result.WriteString(fmt.Sprintf("BSON Document: %+v\n", bsonDoc))

		case 0x01:
			// Type 1: BSON 文档数组，0个或多个BSON对象
			// result.WriteString("Section Type 1 (Multiple BSON Documents)\n")

			// 读取文档序列总长度
			var sequenceLength int32
			binary.Read(reader, binary.LittleEndian, &sequenceLength)
			// result.WriteString(fmt.Sprintf("Document Sequence Length: %d bytes\n", sequenceLength))

			// 读取标识符（C字符串）
			identifier := readCString(reader)
			result.WriteString(fmt.Sprintf("Sequence Identifier: %s\n", identifier))

			// 计算剩余大小
			remainingSize := int(sequenceLength) - len(identifier) - 1 - 4 // 减去标识符、null终止符和长度字段

			// 读取序列中的所有文档
			var documents []map[string]interface{}
			for remainingSize > 0 {
				doc, err := readBSON(reader)
				if err != nil {
					result.WriteString(fmt.Sprintf("Error parsing BSON document in sequence: %v\n", err))
					return "", result.String()
				}
				documents = append(documents, doc)

				// 更新剩余大小
				docSize := getBSONSize(doc)
				remainingSize -= docSize
			}

			result.WriteString(fmt.Sprintf("Documents in Sequence: %+v\n", documents))

		default:
			result.WriteString(fmt.Sprintf("Unknown Section Type: 0x%X\n", sectionType))
			return "", result.String()
		}
	}

	// 如果有校验和，读取并验证
	if hasChecksum {
		var checksum uint32
		binary.Read(reader, binary.LittleEndian, &checksum)
		result.WriteString(fmt.Sprintf("Checksum: 0x%X\n", checksum))
	}

	// 生成操作摘要
	if commandName != "" {
		switch commandName {
		case "insert":
			result.WriteString(fmt.Sprintf("操作摘要: 向 %s.%s 插入文档\n", database, collection))
		case "update":
			result.WriteString(fmt.Sprintf("操作摘要: 更新 %s.%s 中的文档\n", database, collection))
		case "delete":
			result.WriteString(fmt.Sprintf("操作摘要: 从 %s.%s 删除文档\n", database, collection))
		case "find":
			result.WriteString(fmt.Sprintf("操作摘要: 查询 %s.%s 中的文档\n", database, collection))
		case "findAndModify":
			result.WriteString(fmt.Sprintf("操作摘要: 查找并修改 %s.%s 中的文档\n", database, collection))
		case "getMore":
			result.WriteString(fmt.Sprintf("操作摘要: 从 %s.%s 获取更多文档 (游标ID: %v)\n",
				database, collection, commandValue))
		case "count":
			result.WriteString(fmt.Sprintf("操作摘要: 计算 %s.%s 中的文档数量\n", database, collection))
		case "aggregate":
			result.WriteString(fmt.Sprintf("操作摘要: 在 %s.%s 上执行聚合操作\n", database, collection))
		default:
			result.WriteString(fmt.Sprintf("操作摘要: 在 %s.%s 上执行 %s 操作\n",
				database, collection, commandName))
		}
	}

	return commandName, result.String()
}

// 读取C风格字符串（以null结尾）
func readCString(reader *bytes.Reader) string {
	var bytes []byte
	for {
		b, err := reader.ReadByte()
		if err != nil || b == 0 {
			break
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

// 获取BSON文档的大小
func getBSONSize(doc map[string]interface{}) int {
	bytes, _ := bson.Marshal(doc)
	return len(bytes)
}

// 读取 BSON 文档并解析
func readBSON(reader *bytes.Reader) (map[string]interface{}, error) {
	// 检查是否还有足够的数据可读
	if reader.Len() < 4 {
		return nil, fmt.Errorf("数据不足以读取BSON长度: 只有%d字节", reader.Len())
	}

	// 先读取 BSON 头部，包含文档长度
	var bsonLength int32
	if err := binary.Read(reader, binary.LittleEndian, &bsonLength); err != nil {
		return nil, fmt.Errorf("读取BSON长度失败: %v", err)
	}

	// 检查长度是否合理
	if bsonLength <= 4 || bsonLength > 16*1024*1024 { // 16MB是MongoDB文档大小限制
		return nil, fmt.Errorf("BSON长度无效: %d", bsonLength)
	}

	// 检查是否有足够的数据可读
	if reader.Len() < int(bsonLength)-4 {
		return nil, fmt.Errorf("数据不足以读取完整BSON文档: 需要%d字节，但只有%d字节",
			bsonLength-4, reader.Len())
	}

	// 读取完整 BSON 文档
	bsonBytes := make([]byte, bsonLength)

	// 将长度写入前4字节
	binary.LittleEndian.PutUint32(bsonBytes[:4], uint32(bsonLength))

	// 读取剩余部分
	n, err := reader.Read(bsonBytes[4:])
	if err != nil {
		return nil, fmt.Errorf("读取BSON内容失败: %v", err)
	}

	if n != int(bsonLength)-4 {
		return nil, fmt.Errorf("BSON内容不完整: 读取了%d字节，期望%d字节", n, bsonLength-4)
	}

	// 解析 BSON
	var bsonDoc map[string]interface{}
	err = bson.Unmarshal(bsonBytes, &bsonDoc)
	if err != nil {
		// 尝试解析为 bson.D 类型
		var bsonD bson.D
		err2 := bson.Unmarshal(bsonBytes, &bsonD)
		if err2 != nil {
			return nil, fmt.Errorf("解析BSON失败: %v", err)
		}

		// 将 bson.D 转换为 map
		bsonDoc = make(map[string]interface{})
		for _, elem := range bsonD {
			bsonDoc[elem.Key] = elem.Value
		}
	}

	return bsonDoc, nil
}
