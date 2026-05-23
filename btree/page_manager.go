package btree

import (
	"io"
	"os"
)

type pageManager struct {
	file *os.File
}

func newPageManager(fd *os.File) *pageManager {
	return &pageManager{fd}
}

func (nm *pageManager) allocate() (uint32, error) {
	// create a new empty page, set the
	buf := make([]byte, 4096)
	offset, err := nm.file.Seek(0, io.SeekEnd)
	nm.file.Write(buf)
	return uint32(offset / 4096), err
}

func (nm *pageManager) read(pageNum uint32) ([]byte, error) {
	// given a page number
	start := int64(pageNum * PAGE_SIZE)
	buf := make([]byte, 4096)
	// read the bytes of tha page into the buffer
	_, err := nm.file.ReadAt(buf, start)
	return buf, err
}

func (nm *pageManager) write(pageNum uint32, buf []byte) error {
	// given a page number and 4096 bytes
	start := int64(pageNum * PAGE_SIZE)
	// write the bytes for that page
	_, err := nm.file.WriteAt(buf, start)
	return err
}

func (nm *pageManager) len() int {
	var pageNum uint32
	for {
		_, err := nm.read(pageNum)
		if err != nil {
			break
		}
		pageNum++
	}
	return int(pageNum)
}
