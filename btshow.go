package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"
)

const (
	protocolID    = 0x41727101980 // Magic constant
	actionConnect = 0             // Connect action
)

type TrackerClient struct {
	conn           *net.UDPConn
	connectionID   int64
	connectedSince *time.Time
}

type ConnectRequest struct {
	transactionId int32
	raw           []byte
}

func generateTransactionID() int32 {
	return rand.Int31()
}

func NewTrackerClient(host string) TrackerClient {
	trackerAddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatalf("Failed to resolve tracker address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, trackerAddr)
	if err != nil {
		log.Fatalf("Failed to dial tracker: %v", err)
	}

	return TrackerClient{
		conn:           conn,
		connectionID:   0,
		connectedSince: nil,
	}
}

func (t *TrackerClient) connect() {
	if t.connectedSince != nil && time.Since(*t.connectedSince).Seconds() < 60 {
		return // already connected. According to spec, this is reusable for a minute
	}
	connectRequest := NewConnectRequest()
	// Send the connect request
	if _, err := t.conn.Write(connectRequest.raw); err != nil {
		log.Fatalf("Failed to send connect request: %v", err)
	}

	// Receive the response
	response := make([]byte, 16) // Expected response size
	n, err := t.conn.Read(response)
	if err != nil {
		log.Fatalf("Failed to receive response: %v", err)
	}
	if n < 16 {
		log.Fatalf("Response too short")
	}

	// Parse response
	respBuffer := bytes.NewReader(response)

	var recvAction, recvTransactionID int32
	var connectionID int64

	if err := binary.Read(respBuffer, binary.BigEndian, &recvAction); err != nil {
		log.Fatalf("Failed to read action from response: %v", err)
	}
	if recvAction != actionConnect {
		log.Fatalf("Unexpected action in response: %d", recvAction)
	}

	if err := binary.Read(respBuffer, binary.BigEndian, &recvTransactionID); err != nil {
		log.Fatalf("Failed to read transaction ID from response: %v", err)
	}
	if recvTransactionID != connectRequest.transactionId {
		log.Fatalf("Transaction ID mismatch")
	}

	if err := binary.Read(respBuffer, binary.BigEndian, &connectionID); err != nil {
		log.Fatalf("Failed to read connection ID from response: %v", err)
	}

	log.Printf("Received connection ID: %d\n", connectionID)
	t.connectionID = connectionID
	now := time.Now()
	t.connectedSince = &now

}

func (t *TrackerClient) close() error {
	return t.conn.Close()
}

func NewConnectRequest() *ConnectRequest {
	transactionID := generateTransactionID()
	buf := make([]byte, 16)

	// Write protocol ID as a 64-bit integer
	binary.BigEndian.PutUint64(buf[0:], protocolID)
	binary.BigEndian.PutUint32(buf[8:], actionConnect)
	binary.BigEndian.PutUint32(buf[12:], uint32(transactionID))

	return &ConnectRequest{transactionId: transactionID, raw: buf}
}

func main() {
	client := NewTrackerClient("epider.me:6969")
	defer client.close()

	client.connect()
}
