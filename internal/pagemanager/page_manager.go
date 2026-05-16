package pagemanager

import (
	"fmt"
	"io"
	"os"
)

type PageManager struct {
	file        *os.File
	sync        bool // fsync durability
	maxPageSize uint32
}

func NewPageManager(fd *os.File, maxPageSize uint32, sync bool) *PageManager {
	return &PageManager{fd, sync, maxPageSize}
}

func (pm *PageManager) Allocate() (uint32, error) {
	// create a new empty page, set the
	buf := make([]byte, 4096)
	offset, err := pm.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := pm.file.Write(buf); err != nil {
		return 0, err
	}
	return uint32(offset / 4096), nil
}

func (pm *PageManager) Read(pageNum uint32) ([]byte, error) {
	// given a page number
	start := int64(pageNum * pm.maxPageSize)
	buf := make([]byte, 4096)
	// read the bytes of tha page into the buffer
	_, err := pm.file.ReadAt(buf, start)
	return buf, err
}

func (pm *PageManager) Write(pageNum uint32, buf []byte) error {
	// given a page number and 4096 bytes
	start := int64(pageNum * pm.maxPageSize)
	// write the bytes for that page
	_, err := pm.file.WriteAt(buf, start)
	// fsync durability
	if pm.sync {
		if err := pm.file.Sync(); err != nil {
			return fmt.Errorf("failed to fsync page to disk: %w", err)
		}
	}
	return err
}

func (pm *PageManager) len() int {
	info, err := pm.file.Stat()
	if err != nil {
		panic(err)
	}
	return int(info.Size() / int64(pm.maxPageSize))
}
