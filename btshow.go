package main

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"
)

const (
	protocolID    = 0x41727101980
	actionConnect = 0
	actionScrape  = 2
	actionError   = 3
)

type InfoHash = [20]byte

type TrackerClient struct {
	conn           *net.UDPConn
	connectionID   uint64
	connectedSince *time.Time
}

type Request struct {
	transactionId uint32
	action        uint32
	raw           []byte
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

func (t *TrackerClient) connect() error {
	if t.connectedSince != nil && time.Since(*t.connectedSince).Seconds() < 60 {
		return nil // already connected. According to spec, this is reusable for a minute
	}
	connectRequest := NewConnectRequest()
	if _, err := t.conn.Write(connectRequest.raw); err != nil {
		log.Fatalf("Failed to send connect request: %v", err)
	}
	response, err := t.read(connectRequest)
	if err != nil {
		return err
	}
	connectionID := binary.BigEndian.Uint64(response[8:])
	log.Printf("Received connection ID: %d\n", connectionID)
	t.connectionID = connectionID
	now := time.Now()
	t.connectedSince = &now
	return nil
}

func (t *TrackerClient) read(request *Request) ([]byte, error) {
	var expectedSize int
	switch request.action {
	case actionConnect:
		expectedSize = 16
	default:
		panic("invalid request value")
	}
	response := make([]byte, expectedSize) // Expected response size
	n, err := t.conn.Read(response)
	if err != nil {
		log.Fatalf("Failed to receive response: %v", err)
		return nil, err
	}
	if n < expectedSize {
		log.Fatalf("Response too short")
	}

	recvAction := binary.BigEndian.Uint32(response[0:])
	recvTransactionID := binary.BigEndian.Uint32(response[4:])
	// Parse response
	if recvAction != request.action {
		log.Fatalf("Unexpected action in response: %d (wanted %d)", recvAction, request.action)
	}

	if recvTransactionID != request.transactionId {
		log.Fatalf("Transaction ID mismatch")
	}

	return response, nil
}

func (t *TrackerClient) close() error {
	return t.conn.Close()
}

func NewConnectRequest() *Request {
	transactionID := rand.Uint32()
	buf := make([]byte, 16)

	// Write protocol ID as a 64-bit integer
	binary.BigEndian.PutUint64(buf[0:], protocolID)
	binary.BigEndian.PutUint32(buf[8:], actionConnect)
	binary.BigEndian.PutUint32(buf[12:], transactionID)

	return &Request{transactionId: transactionID, raw: buf, action: actionConnect}
}

func main() {
	client := NewTrackerClient("epider.me:6969")
	defer client.close()

	client.connect()
}
