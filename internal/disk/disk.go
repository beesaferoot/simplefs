package disk

import (
	"fmt"
	"os"
	"syscall"
)

/*
Disk emulator
*/
const BLOCK_SIZE = 4096

type Disk struct {
	Name           string // File name of disk image
	FileDescriptor int    // File descriptor of disk image
	Blocks         uint32 // Number of blocks in disk image
	Reads          uint32 // Number of total reads performed on disk
	Writes         uint32 // Number of total writes performed on disk
	Mounts         uint32 // Number of total mounts
}

// Open disk image
// path: Path to disk image
// nblocks: Number of blocks in disk image
func (d *Disk) Open(path string, nblocks int) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("Unable to open %s: %s", path, err.Error())
	}
	if err = file.Truncate(int64(nblocks * BLOCK_SIZE)); err != nil {
		return fmt.Errorf("Unable to open %s: %s", path, err.Error())
	}
	d.FileDescriptor = int(file.Fd())
	d.Name = path
	d.Blocks = uint32(nblocks)
	d.Reads = 0
	d.Writes = 0
	return nil
}

// Close underlying disk image file
func (d *Disk) Close() error {
	return syscall.Close(d.FileDescriptor)
}

// Return size of disk (in terms of blocks)
func (d *Disk) Size() uint32 {
	return d.Blocks
}

// Return whether or not disk is mounted
func (d *Disk) Mouted() bool {
	return d.Mounts > 0
}

// Decrement mounts
func (d *Disk) UnMount() {
	if d.Mounts > 0 {
		d.Mounts--
	}
}

// Increment mounts
func (d *Disk) Mount() {
	d.Mounts++
}

// Read block from disk
//
// blocknum: Block to read from
//
// data: Buffer to read into
func (d *Disk) Read(blocknum int, data []byte) error {
	err := d.sanityCheck(blocknum)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(d.FileDescriptor), "image file")
	if _, err := file.Seek(int64(blocknum*BLOCK_SIZE), os.SEEK_SET); err != nil {
		return fmt.Errorf("Unable to seek %d: %s", blocknum, err.Error())
	}
	read_bytes, err := file.Read(data)
	if err != nil {
		return fmt.Errorf("Unable to read %d: %s", blocknum, err.Error())
	}
	if read_bytes != BLOCK_SIZE {
		return fmt.Errorf("Unable to read %d", blocknum)
	}
	d.Reads++
	return nil
}

// Write block to disk
//
// blocknum: Block to write to
//
// data: Buffer to write from
func (d *Disk) Write(blocknum int, data []byte) error {

	if BLOCK_SIZE < len(data) {
		return fmt.Errorf("Unable to write to block (%d): size of data greater than %d bytes", blocknum, BLOCK_SIZE)
	}

	err := d.sanityCheck(blocknum)

	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(d.FileDescriptor), "image file")
	if _, err := file.Seek(int64(blocknum*BLOCK_SIZE), os.SEEK_SET); err != nil {
		return fmt.Errorf("Unable to seek block (%d): %s", blocknum, err.Error())
	}

	written_bytes, err := file.Write(data)
	if err != nil {
		return fmt.Errorf("Unable to write to block (%d): %s", blocknum, err.Error())
	}
	if written_bytes > BLOCK_SIZE {
		fmt.Printf("written_bytes = %d\n", written_bytes)
	}
	d.Writes++
	return nil
}

// validate given parameters
// blocknum: Block to operate on
func (d *Disk) sanityCheck(blocknum int) error {
	if blocknum < 0 {
		return fmt.Errorf("blocknum (%d) is negative", blocknum)
	}
	if blocknum >= int(d.Blocks) {
		return fmt.Errorf("blocknum (%d) is too large", blocknum)
	}
	return nil
}
