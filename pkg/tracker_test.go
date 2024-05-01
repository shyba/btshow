package pkg

import (
	"crypto/rand"
	"reflect"
	"testing"
)

func TestNewConnectRequest(t *testing.T) {
	req := NewConnectRequest()
	if req.action != actionConnect {
		t.Errorf("wrong action")
	}
	if len(req.raw) != 16 {
		t.Errorf("wrong serialized size")
	}
	anotherReq := NewConnectRequest()
	if req.transactionId == anotherReq.transactionId {
		t.Errorf("transaction id is not unique")
	}
	if reflect.DeepEqual(req.raw, anotherReq.raw) {
		t.Errorf("serialization is not unique")
	}
}

func TestNewScrapeRequest(t *testing.T) {
	var exampleInfohashes []InfoHash = make([]InfoHash, 3)
	for idx, infohash := range exampleInfohashes {
		rand.Read(infohash[:])
		exampleInfohashes[idx] = infohash
	}
	type args struct {
		connectionID uint64
		infohashes   []InfoHash
	}
	tests := []struct {
		name string
		args args
		want *Request
	}{
		{"single", args{uint64(42), []InfoHash{exampleInfohashes[0]}}, &Request{transactionId: uint32(12), action: actionScrape}},
		{"multi", args{uint64(12), exampleInfohashes}, &Request{transactionId: uint32(12), action: actionScrape}},
	}
	var lastTid *uint32 = nil
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewScrapeRequest(tt.args.connectionID, tt.args.infohashes...)
			if len(got.raw) != 16+20*len(tt.args.infohashes) {
				t.Errorf("wrong length. Expected %d, got %d", 16+20*len(tt.args.infohashes), len(got.raw))
			}
			if lastTid != nil && got.transactionId == *lastTid {
				t.Errorf("transaction id not unique")
			}
			if got.action != tt.want.action {
				t.Errorf("wrong action")
			}
			lastTid = &got.transactionId
			for idx, infohash := range tt.args.infohashes {
				if reflect.DeepEqual(infohash, InfoHash{}) {
					t.Errorf("empty infohash at %d", idx)
				}
				if !reflect.DeepEqual(got.raw[16+20*idx:16+20*idx+20], infohash[0:20]) {
					t.Errorf("missing infohash %v", infohash[0:20])
				}
			}
		})
	}
}
