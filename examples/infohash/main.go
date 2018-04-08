package main

// ./infohash 183.230.252.42 63921 9daa6500196e410fcdb234d1ceb1b73c03fff25c f4a41be033406ac51f2d2d847c52d5f50d227e9d

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/IncSW/go-bencode"
	"io"
	// "io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"time"
)

const (
	BLOCK = 22808
)

var r = rand.New(rand.NewSource(time.Now().Unix()))

// SendExtMessage 发送数据
func SendExtMessage(conn net.Conn, data []byte) (n int, err error) {
	var (
		bid int64 = 20
		eid int64
	)
	return sendExtMessage(conn, bid, eid, data)
}

func sendExtMessage(conn net.Conn, bid, eid int64, data []byte) (n int, err error) {
	length := len(data) + 2
	message := make([]byte, 4+length)
	binary.BigEndian.PutUint32(message[:4], uint32(length))
	message[4] = byte(bid)
	message[5] = byte(eid)
	copy(message[6:], data)

	return sendData(conn, message)
}

func requestPieces(conn net.Conn, data []byte) error {
	i, err := bencode.Unmarshal(data)
	if err != nil {
		return err
	}

	fmt.Println(i)

	info := i.(map[string]interface{})
	metadataSize := info["metadata_size"].(int64)

	m := info["m"].(map[string]interface{})
	utMetadata := m["ut_metadata"].(int64)

	piecesNum := metadataSize / BLOCK
	if metadataSize%BLOCK > 0 {
		piecesNum++
	}

	var bid int64 = 20
	var eid = utMetadata
	for i := 0; int64(i) < piecesNum; i++ {

		msg, _ := bencode.Marshal(map[string]interface{}{
			"msg_type": 0,
			"piece":    i,
		})

		sendExtMessage(conn, bid, eid, msg)
	}
	return nil

}

func sendData(conn net.Conn, data []byte) (n int, err error) {
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	fmt.Println("send", data)
	return conn.Write(data)
}

// ReadLength 读取指定长度的数据包
func ReadLength(conn net.Conn, length int, buf *bytes.Buffer) error {
	n, err := io.CopyN(buf, conn, int64(length))
	if err != nil || n != int64(length) {
		fmt.Println(err, n, buf.Next(int(n)))
		return errors.New("read error")
	}
	return nil
}

// ReadPacket 根据协议读包
func ReadPacket(conn net.Conn, buf *bytes.Buffer) error {
	if err := ReadLength(conn, 4, buf); err != nil {
		return err
	}

	b := buf.Next(4)
	length := binary.BigEndian.Uint32(b)
	fmt.Println("length", b, length)
	err := ReadLength(conn, int(length), buf)
	return err
}

func GenExtShakeHand() []byte {
	metadata := map[string]interface{}{
		"m": map[string]interface{}{
			"ut_metadata": 1,
		},
	}
	b, _ := bencode.Marshal(metadata)
	return b
}

func GenShakeHand(id string, hash string) []byte {
	message := make([]byte, 68)
	message[0] = 19
	copy(message[1:20], []byte("BitTorrent protocol"))
	message[25] = 0x10
	message[27] = 1

	infohash, _ := hex.DecodeString(hash)
	peerID, _ := hex.DecodeString(id)

	copy(message[28:48], infohash)
	copy(message[48:68], peerID)

	return message
}

func readBytes(c net.Conn) {
	b := make([]byte, 1024)

	for {
		n, err := c.Read(b)
		if err != nil {
			panic(err)
		}

		fmt.Println("recv", b[:n])
	}
}

func main() {
	ip, port, id, hash := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	addr := fmt.Sprintf("%s:%s", ip, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		panic(err)
	}

	shakehand := GenShakeHand(id, hash)
	sendData(conn, shakehand)

	data := new(bytes.Buffer)
	err = ReadLength(conn, 68, data)
	if err != nil {
		panic(err)
	}
	ret := data.Next(68)
	fmt.Println("recv <=", ret, ret[25]&0x10)

	data.Reset()

	msg := GenExtShakeHand()
	SendExtMessage(conn, msg)

	for {
		fmt.Println("=========packet begin===========")
		err = ReadPacket(conn, data)
		if err != nil {
			panic(err)
		}

		if data.Len() == 0 {
			fmt.Println("recv <= empty")
			continue
		}

		bid, _ := data.ReadByte()
		eid, _ := data.ReadByte()

		fmt.Println(bid, eid)
		if bid == 20 {
			if eid == 0 {
				// b, err := ioutil.ReadAll(conn)
				// fmt.Println(err, b)
				requestPieces(conn, data.Bytes())

			} else {

				in := data.Bytes()
				i, err := bencode.Unmarshal(in)
				b, err := bencode.Marshal(i)
				fmt.Println(i, err)

				i, err = bencode.Unmarshal(in[len(b):])
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(string(i.(map[string]interface{})["name"].([]byte)))
				}
			}
			goto reset
		}

		fmt.Println(data.Bytes())

	reset:
		fmt.Println("=========packet end===========")
		fmt.Println()
		data.Reset()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c
	conn.Close()
}
