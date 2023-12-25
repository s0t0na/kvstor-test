package main

import (
	"fmt"
	"net"
	"testing"
	"time"
)

const testKVStorPort = 8666

func TestKVStorSetGet(t *testing.T) {
	kv := NewKVStor()

	// Test SET and GET operations
	kv.Set("key1", "value1", 10)
	val, ok := kv.Get("key1")
	if !ok {
		t.Error("expected to find key1 in KVStor, but it wasn't found.")
	}

	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	// Test SET with expiration
	kv.Set("key2", "value2", 1)
	time.Sleep(2 * time.Second) // Wait for key2 to expire
	_, ok = kv.Get("key2")
	if ok {
		t.Error("expected key2 to be expired, but it was found.")
	}
}

func TestKVStorDelete(t *testing.T) {
	kv := NewKVStor()

	// Test DELETE operation
	kv.Set("key3", "value3", 10)
	panicOnError(kv.Delete("key3"))

	_, ok := kv.Get("key3")
	if ok {
		t.Error("expected key3 to be deleted, but it was found.")
	}
}

func TestHandleRequestSet(t *testing.T) {

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", testKVStorPort))
	if err != nil {
		t.Fatalf("error connecting to KVStor: %v", err)
	}
	defer conn.Close()

	// Test SET
	conn.Write([]byte("set key4 value4 10\n"))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	responseStr := string(response[:n])
	expectedResponse := "set key: \033[35mkey4\033[0m with value \033[35mvalue4\033[0m and TTL \033[35m10s\033[0m\n"

	if responseStr != expectedResponse {
		t.Errorf("expected response: %s, got: %s", expectedResponse, responseStr)
	}
}

func TestHandleRequestGet(t *testing.T) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", testKVStorPort))
	if err != nil {
		t.Fatalf("error connecting to KVStor: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("set key5 value5 3600\n"))
	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	// Test GET
	conn.Write([]byte("get key5\n"))
	response = make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	responseStr := string(response[:n])
	expectedResponse := "key: \033[35mkey5\033[0m has value \u001B[35mvalue5\033[0m\n"

	if responseStr != expectedResponse {
		t.Errorf("expected response: %s, got: %s", expectedResponse, responseStr)
	}
}

func TestHandleRequestDelete(t *testing.T) {

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", testKVStorPort))
	if err != nil {
		t.Fatalf("error connecting to KVStor: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("set key6 value6 3600\n"))
	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	// Test DELETE
	conn.Write([]byte("delete key6\n"))
	response = make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	responseStr := string(response[:n])
	expectedResponse := "key: \033[35mkey6\033[0m was deleted\n"

	if responseStr != expectedResponse {
		t.Errorf("expected response: %s, got: %s", expectedResponse, responseStr)
	}
}

func TestHandleRequestInvalidCommand(t *testing.T) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", testKVStorPort))
	if err != nil {
		t.Fatalf("error connecting to KVStor: %v", err)
	}
	defer conn.Close()

	// Test handling an invalid command
	conn.Write([]byte("INVALID_COMMAND\n"))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("error reading response: %v", err)
	}

	responseStr := string(response[:n])
	expectedResponse := usage()

	if responseStr != expectedResponse {
		t.Errorf("expected response: %s, got: %s", expectedResponse, responseStr)
	}
}
