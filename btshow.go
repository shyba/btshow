package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
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

type InfohashStat struct {
	seeders   uint32
	completed uint32
	leechers  uint32
}

type ScrapeResponse = map[InfoHash]InfohashStat

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
	response, err := t.sendRequest(NewConnectRequest())
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

func (t *TrackerClient) scrape(infohashes ...InfoHash) (ScrapeResponse, error) {
	if err := t.connect(); err != nil {
		return nil, err
	}
	responseBytes, err := t.sendRequest(NewScrapeRequest(t.connectionID, infohashes...))
	if err != nil {
		return nil, err
	}

	scrapeResponse := make(ScrapeResponse)
	for idx, infohash := range infohashes {
		scrapeResponse[infohash] = InfohashStat{
			seeders:   binary.BigEndian.Uint32(responseBytes[8+idx*12:]),
			completed: binary.BigEndian.Uint32(responseBytes[8+idx*12+8:]),
			leechers:  binary.BigEndian.Uint32(responseBytes[8+idx*12+4:]),
		}
	}

	return scrapeResponse, nil
}

func (t *TrackerClient) sendRequest(request *Request) ([]byte, error) {
	// todo: retry
	if _, err := t.conn.Write(request.raw); err != nil {
		log.Fatalf("Failed to send connect request: %v", err)
	}
	return t.read(request)
}

func (t *TrackerClient) read(request *Request) ([]byte, error) {
	var expectedSize int
	switch request.action {
	case actionConnect:
		expectedSize = 16
	case actionScrape:
		total := (len(request.raw) - 16) / 20
		expectedSize = 8 + 12*total
	default:
		panic("invalid request value")
	}
	response := make([]byte, expectedSize) // Expected response size
	n, err := t.conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %v", err)
	}
	if n < expectedSize {
		log.Fatalf("Response too short")
	}

	recvAction := binary.BigEndian.Uint32(response[0:])
	recvTransactionID := binary.BigEndian.Uint32(response[4:])
	// Parse response
	if recvAction == actionError {
		return nil, fmt.Errorf("received error: %s", response[8:])
	}
	if recvAction != request.action {
		return nil, fmt.Errorf("unexpected action in response: %d (wanted %d)", recvAction, request.action)
	}

	if recvTransactionID != request.transactionId {
		return nil, fmt.Errorf("transaction ID mismatch")
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

func NewScrapeRequest(connectionID uint64, infohashes ...InfoHash) *Request {
	transactionID := rand.Uint32()
	buf := make([]byte, 16+20*len(infohashes))

	// Write protocol ID as a 64-bit integer
	binary.BigEndian.PutUint64(buf[0:], connectionID)
	binary.BigEndian.PutUint32(buf[8:], actionScrape)
	binary.BigEndian.PutUint32(buf[12:], transactionID)
	for idx, infohash := range infohashes {
		copy(buf[(idx*20+16):], infohash[:])
	}

	return &Request{transactionId: transactionID, raw: buf, action: actionScrape}

}

func parseInfohash(infohash string) InfoHash {
	val, err := hex.DecodeString(infohash)
	if err != nil {
		panic(err)
	}
	return InfoHash(val[0:20])
}

func printInfohashResponse(response ScrapeResponse) {
	for infohash := range response {
		fmt.Println(hex.EncodeToString(infohash[:]))
		stat := response[infohash]
		fmt.Printf("Completed: %d\n", stat.completed)
		fmt.Printf("Leechers: %d\n", stat.leechers)
		fmt.Printf("Seeders: %d\n", stat.seeders)
	}
}

func main() {
	if len(os.Args) < 3 {
		println("Usage: btshow <host> {<infohash>,...}+")
		return
	}
	client := NewTrackerClient(os.Args[1])
	defer client.close()

	args := os.Args[2:]
	infohashes := make([]InfoHash, len(args))
	for idx, arg := range args {
		infohashes[idx] = parseInfohash(arg)
	}

	resp, err := client.scrape(infohashes...)
	printInfohashResponse(resp)
	if err != nil {
		panic(err)
	}
}
