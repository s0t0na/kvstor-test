package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const KVStorProto string = "tcp"
const KVStorPort uint16 = 8666
const CleanupCheckTimeout = time.Second * 60 // cleanup runs every minute.
const DefaultRecordTTL uint64 = 3600

type KVStor struct {
	payload map[string]Record
	lock    sync.Mutex
}

type Record struct {
	Key   string
	Value []byte

	Timestamp  uint64
	TimeToLive uint64

	// todo: this is to prevent overflows
	KeySize   uint64
	ValueSize uint64
	CheckSum  []byte
}

// NewKVStor create new KVStor instance.
func NewKVStor() *KVStor {
	return &KVStor{
		payload: make(map[string]Record),
	}
}

// AutoCleanObsolete checks all records every minute and deletes obsolete ones. May be expensive though.
func (k *KVStor) AutoCleanObsolete() {
	for {
		time.Sleep(CleanupCheckTimeout)

		k.lock.Lock()
		for key, rec := range k.payload {
			if (rec.Timestamp + rec.TimeToLive) < uint64(time.Now().Unix()) {
				delete(k.payload, key)
			}
		}

		k.lock.Unlock()
	}
}

// Delete deletes a key if it exists.
func (k *KVStor) Delete(key string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	_, ok := k.payload[key]
	if ok {
		delete(k.payload, key)
		return nil
	}

	return errors.New("no such key")
}

// Get gets a key value if the key exists.
func (k *KVStor) Get(key string) (string, bool) {
	k.lock.Lock()
	defer k.lock.Unlock()

	rec, ok := k.payload[key]
	if !ok {
		return "", false
	}

	if (rec.Timestamp + rec.TimeToLive) < uint64(time.Now().Unix()) {
		delete(k.payload, key)
		return "", false
	}

	return string(rec.Value), true
}

// Set sets the provided key with the provided value. In case if TTL is provided also sets TTL, otherwise TTL is 3600 seconds.
func (k *KVStor) Set(key string, value string, ttl uint64) {
	k.lock.Lock()
	defer k.lock.Unlock()

	k.payload[key] = Record{
		Key:        key,
		Value:      []byte(value),
		Timestamp:  uint64(time.Now().Unix()),
		TimeToLive: ttl,
	}
}

// HandleRequest handles user request to kvstor.
func HandleRequest(conn net.Conn, k *KVStor) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		request := scanner.Text()
		var fields = make([]string, 4)

		for i, v := range strings.Fields(request) {
			fields[i] = v
		}

		action := fields[0]
		key := fields[1]

		switch action {
		case "set":
			value := fields[2]
			var ttl int64

			if fields[3] != "" {
				ttl, _ = strconv.ParseInt(fields[3], 10, 64)
			} else {
				ttl = int64(DefaultRecordTTL)
			}

			k.Set(key, value, uint64(ttl))
			_, err := conn.Write([]byte(fmt.Sprintf("set key: \033[35m%s\033[0m with value \033[35m%s\033[0m and TTL \033[35m%ds\033[0m\n", key, value, ttl)))
			if err != nil {
				writeUsageOrHandleUnexpectedError(conn, err)
			}
		case "get":
			value, ok := k.Get(key)
			if ok {
				_, err := conn.Write([]byte(fmt.Sprintf("key: \033[35m%s\033[0m has value \u001B[35m%s\033[0m\n", key, value)))
				if err != nil {
					writeUsageOrHandleUnexpectedError(conn, err)
				}
			} else {
				_, err := conn.Write([]byte(fmt.Sprintf("key: \033[35m%s\033[0m doesnt't exist or expired\n", key)))
				if err != nil {
					writeUsageOrHandleUnexpectedError(conn, err)
				}
			}
		case "delete":
			err := k.Delete(key)
			if err != nil {
				_, err = conn.Write([]byte(fmt.Sprintf("deletion error: %v\n", err)))
				if err != nil {
					writeUsageOrHandleUnexpectedError(conn, err)
				}
				continue
			}

			_, err = conn.Write([]byte(fmt.Sprintf("key: \033[35m%s\033[0m was deleted\n", key)))
			if err != nil {
				writeUsageOrHandleUnexpectedError(conn, err)
			}
		default:
			writeUsageOrHandleUnexpectedError(conn, nil)
		}
	}

}

func main() {
	kvstor := NewKVStor()

	// We shall clean up obsolete records in a separate goroutine.
	go kvstor.AutoCleanObsolete()

	server, err := net.Listen(KVStorProto, fmt.Sprintf(":%d", KVStorPort))
	if err != nil {
		panic(fmt.Sprintf("could not start listening port %d: %v", KVStorPort, err))
	}

	defer server.Close()

	fmt.Printf("KVStor is now listening on port :%d\n\n", KVStorPort)
	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Printf("an error occured: %v\n\n", err)
			continue
		}

		go HandleRequest(conn, kvstor)
	}

}

// usage explains how to use kvstor.
func usage() string {
	return "Usage: get <key> | set <key> <value> [TTL in seconds] | delete <key>\n"
}

// writeUsageOrHandleUnexpectedError writes usage() output to the stream or panics is Write() was unsuccessful.
func writeUsageOrHandleUnexpectedError(conn net.Conn, err error) {
	if err != nil {
		panic(err)
	}

	_, err2 := conn.Write([]byte(usage()))
	if err2 != nil {
		panic("something weird has happened.")
	}
}

// TODO: saving data to disk would not hurt though.
