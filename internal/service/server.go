// Package mcstatus 提供了查詢 Minecraft 伺服器狀態的功能
package mcstatus

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

// ServerStatus 定義了從 Minecraft 伺服器接收到的狀態信息結構
type ServerStatus struct {
	Version struct {
		Name     string `json:"name"`     // 伺服器版本名稱
		Protocol int    `json:"protocol"` // 伺服器協議版本
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample"`
	} `json:"players"`
	Description struct {
		Text  string `json:"text"` // 伺服器描述文本
		Extra []struct {
			Text  string `json:"text"`            // 額外描述文本
			Color string `json:"color,omitempty"` // 文本顏色（可選）
		} `json:"extra,omitempty"` // 額外描述信息（可選）
	} `json:"description"`
	Favicon string `json:"favicon"` // 伺服器圖標（Base64 編碼）
}

// PacketBuffer 用於構建網絡數據包
type PacketBuffer struct {
	buffer bytes.Buffer
}

// NewPacketBuffer 創建一個新的 PacketBuffer 實例
func NewPacketBuffer() *PacketBuffer {
	return &PacketBuffer{}
}

// WriteVarInt 將一個 32 位整數編碼為 VarInt 格式並寫入緩衝區
// VarInt 是一種可變長度的整數編碼方式，用於在較少的字節中表示小的數值
func (pb *PacketBuffer) WriteVarInt(val int32) error {
	// 將有符號整數轉換為無符號整數
	// 這是因為 VarInt 編碼實際上是基於無符號整數的
	ux := uint32(val)

	// 持續編碼直到值小於 0x80 (128)
	for ux >= 0x80 {
		// 對於每個字節，除了最後一個：
		// 1. 使用位或運算 (|) 將最高位設置為 1，表示還有後續字節
		// 2. 僅寫入低 7 位作為當前字節的數據位
		if err := pb.buffer.WriteByte(byte(ux) | 0x80); err != nil {
			return err
		}

		// 將值右移 7 位，準備編碼下一個字節
		ux >>= 7
	}

	// 寫入最後一個字節
	// 這個字節的最高位是 0，表示 VarInt 的結束
	return pb.buffer.WriteByte(byte(ux))
}

// WriteString 寫入一個字符串到緩衝區
func (pb *PacketBuffer) WriteString(s string) error {
	if err := pb.WriteVarInt(int32(len(s))); err != nil {
		return err
	}
	_, err := pb.buffer.WriteString(s)
	return err
}

// WriteUnsignedShort 寫入一個無符號短整數到緩衝區
func (pb *PacketBuffer) WriteUnsignedShort(val uint16) error {
	return binary.Write(&pb.buffer, binary.BigEndian, val)
}

// Bytes 返回緩衝區的字節切片
func (pb *PacketBuffer) Bytes() []byte {
	return pb.buffer.Bytes()
}

// GetServerStatus 查詢指定地址的 Minecraft 伺服器狀態
func GetServerStatus(address string) (*ServerStatus, error) {
	log.Printf("開始查詢伺服器狀態: %s", address)

	// 解析地址和端口
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		portStr = "25565" // 默認 Minecraft 端口
	}
	log.Printf("解析後的地址: %s:%s", host, portStr)

	// 查找端口號
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		return nil, fmt.Errorf("無效的端口: %w", err)
	}

	// 解析 IP 地址
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("無法解析主機名: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("無法找到 IP 地址")
	}
	ip := ips[0]
	log.Printf("解析到的 IP: %s", ip)

	// 建立 TCP 連接
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), portStr), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("連接伺服器失敗: %w", err)
	}
	defer conn.Close()
	log.Println("成功建立連接")

	// 設置連接超時
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// 發送握手包
	if err := sendHandshakePacket(conn, host, uint16(port)); err != nil {
		return nil, fmt.Errorf("發送握手數據包失敗: %w", err)
	}
	log.Println("握手數據包發送成功")

	// 發送狀態請求包
	if err := sendStatusRequestPacket(conn); err != nil {
		return nil, fmt.Errorf("發送狀態請求數據包失敗: %w", err)
	}
	log.Println("狀態請求數據包發送成功")

	// 讀取並解析伺服器回應
	rawResponse, err := readAndParseResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("讀取和解析回應失敗: %w", err)
	}
	log.Printf("收到原始回應：%s", string(rawResponse))

	// 解析 JSON 回應
	var status ServerStatus
	err = json.Unmarshal(rawResponse, &status)
	if err != nil {
		// 如果解析失敗，嘗試使用備用結構
		var fallbackStatus struct {
			Description interface{} `json:"description"`
		}
		if err := json.Unmarshal(rawResponse, &fallbackStatus); err != nil {
			return nil, fmt.Errorf("解析 JSON 響應失敗: %w", err)
		}

		// 根據描述的類型進行處理
		switch desc := fallbackStatus.Description.(type) {
		case string:
			status.Description.Text = desc
		case map[string]interface{}:
			if text, ok := desc["text"].(string); ok {
				status.Description.Text = text
			}
		}
	}

	// 處理可能的 Unicode 轉義序列
	status.Description.Text = unescapeUnicode(status.Description.Text)
	for i := range status.Description.Extra {
		status.Description.Extra[i].Text = unescapeUnicode(status.Description.Extra[i].Text)
	}

	log.Println("成功解析 JSON 響應")

	return &status, nil
}

// readAndParseResponse 從連接中讀取並解析伺服器回應
func readAndParseResponse(conn net.Conn) ([]byte, error) {
	// 讀取數據包長度
	_, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("讀取數據包長度失敗: %w", err)
	}

	// 讀取數據包 ID
	packetID, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("讀取數據包 ID 失敗: %w", err)
	}

	if packetID != 0x00 {
		return nil, fmt.Errorf("無效的數據包 ID: %d", packetID)
	}

	// 讀取 JSON 長度
	jsonLength, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("讀取 JSON 長度失敗: %w", err)
	}

	// 讀取 JSON 數據
	jsonData := make([]byte, jsonLength)
	_, err = io.ReadFull(conn, jsonData)
	if err != nil {
		return nil, fmt.Errorf("讀取 JSON 數據失敗: %w", err)
	}

	return jsonData, nil
}

// readVarInt 從 Reader 中讀取一個變長整數
func readVarInt(r io.Reader) (int32, error) {
	var result int32
	var shift uint
	for {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			return 0, err
		}
		result |= int32(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("VarInt 太長")
		}
	}
	return result, nil
}

// sendHandshakePacket 發送握手數據包
func sendHandshakePacket(conn net.Conn, host string, port uint16) error {
	packet := NewPacketBuffer()
	packet.WriteVarInt(0x00)        // Handshake packet ID
	packet.WriteVarInt(-1)          // Protocol version (-1 for status ping)
	packet.WriteString(host)        // Server address
	packet.WriteUnsignedShort(port) // Server port
	packet.WriteVarInt(1)           // Next state (1 for status)
	return sendPacket(conn, packet.Bytes())
}

// sendStatusRequestPacket 發送狀態請求數據包
func sendStatusRequestPacket(conn net.Conn) error {
	packet := NewPacketBuffer()
	packet.WriteVarInt(0x00) // Status request packet ID
	return sendPacket(conn, packet.Bytes())
}

// sendPacket 發送數據包到連接
func sendPacket(conn net.Conn, data []byte) error {
	packet := NewPacketBuffer()
	packet.WriteVarInt(int32(len(data)))
	packet.buffer.Write(data)
	n, err := conn.Write(packet.Bytes())
	if err != nil {
		return fmt.Errorf("發送數據包失敗: %w", err)
	}
	log.Printf("發送數據包成功，長度: %d 字節", n)
	return nil
}

// unescapeUnicode 函數用於解碼字符串中的 Unicode 轉義序列
func unescapeUnicode(s string) string {
	var buf bytes.Buffer
	for i := 0; i < len(s); {
		if i+5 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
			r, err := strconv.ParseInt(s[i+2:i+6], 16, 32)
			if err == nil {
				buf.WriteRune(rune(r))
				i += 6
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}
