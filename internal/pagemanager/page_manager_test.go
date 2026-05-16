package pagemanager

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

const (
	MockPageSizeForTesting = 4096
	MockSyncFlagForTesting = true
)

func setup(t *testing.T) *PageManager {
	fd, err := os.OpenFile("test.bin", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("error opening file: %v", err)
	}

	return NewPageManager(fd, MockPageSizeForTesting, MockSyncFlagForTesting)
}

func TestPageManagerAllocate(t *testing.T) {
	pm := setup(t)
	latestPageNum, err := mockAllocateTwice(pm)
	if err != nil {
		t.Fatalf("failed to allocate a page: %v", err)
	}

	if latestPageNum != 1 {
		t.Fatalf("page number expected: 1, got: %d", latestPageNum)
	}

	if pm.len() != 2 {
		t.Fatalf("total pages expected: 2, got: %d", pm.len())
	}
}

func TestPageManagerReadWrite(t *testing.T) {
	pm := setup(t)

	mockAllocateTwice(pm)
	pageNum, _ := mockAllocateTwice(pm)

	pageNum = rand.Intn(pageNum + 1)
	t.Logf("attempting to write and then read from page number %d", pageNum)

	buf := make([]byte, 4096)
	target := make([]byte, 4096)
	copy(target[:], "hello")
	copy(buf[:], "hello")

	err := pm.Write(uint32(pageNum), buf)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	buf1, err := pm.Read(uint32(pageNum))
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if res := bytes.Compare(target, buf1); res != 0 {
		t.Fatalf("target and retrieved bytes did not match, expected: %v, got: %v, res: %d", string(target), string(buf1), res)
	}
}

func mockAllocateTwice(pm *PageManager) (int, error) {
	_, err := pm.Allocate()
	if err != nil {
		return 0, err
	}
	pageNum, err := pm.Allocate()
	if err != nil {
		return 0, err
	}
	return int(pageNum), nil
}
