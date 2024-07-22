package fs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"simplefs/internal/disk"
)

const (
	MAGIC_NUMBER       = 0xf0f03410
	INODES_PER_BLOCK   = 128
	POINTERS_PER_INODE = 5
	POINTERS_PER_BLOCK = 1024
)

var (
	enc = binary.LittleEndian
)

type FileSystem interface {
	Debug(*disk.Disk) error
	Format(*disk.Disk) bool
	Mount(*disk.Disk) bool

	Create() (int, error)
	Stat(int) (int, error)
	Remove(int) error

	Write(int, DataBlock) error
	Read(int) (DataBlock, error) 
}

type FS struct {
	disk *disk.Disk
	freeBlockBitMap []uint32 // Hold record of used or unused blocks
	superBlock      SuperBlock
	inodeBlocks     []*InodeBlock
	data            DataBlock
}

// Superblock structure
type SuperBlock struct {
	MagicNumber uint32 // File system magic number
	Blocks      uint32 // Number of blocks in file system
	InodeBlocks uint32 // Number of blocks reserved for inodes in file system
	Inodes      uint32 // Number of inodes in file system
}

type Inode struct {
	Valid    uint32                     // Whether or not inode is valid
	Size     uint32                     // Size of file
	Direct   [POINTERS_PER_INODE]uint32 // Direct pointers
	Indirect uint32                     // Indirect pointer
}

type InodeBlock struct {
	Inodes [INODES_PER_BLOCK]Inode // Inode block
}

type DataBlock struct {
	Data [disk.BLOCK_SIZE]byte // Data blockt
}

func NewFS() FileSystem {
	return &FS{freeBlockBitMap: []uint32{}}
}

func (fs *FS) Debug(dsk *disk.Disk) error {
	var sblock SuperBlock
	var iblocks []*InodeBlock
	// Ready Superblock
	if err := fs.loadSuperBlock(dsk, &sblock); err != nil {
		return err
	}
	fmt.Printf("SuperBlock:\n")
	fmt.Printf("    %d blocks\n", sblock.Blocks)
	fmt.Printf("    %d inode blocks\n", sblock.InodeBlocks)
	fmt.Printf("    %d inodes\n", sblock.Inodes)
	// set inode block size to read
	iblocks = make([]*InodeBlock, sblock.InodeBlocks)
	// Read Inode blocks
	err := fs.loadInodeBlocks(dsk, int(sblock.InodeBlocks), iblocks)
	if err != nil {
		return err
	}

	for _, iblock := range iblocks {
		for id, v := range iblock.Inodes {
			if v.Size > 0 {
				fmt.Printf("inode %d:\n", id)
				fmt.Printf("    size: %d bytes\n", v.Size)
				fmt.Printf("    direct blocks: %v\n", v.Direct)
				fmt.Printf("    indirect blocks: %v\n", v.Indirect)
			}
		}

	}
	fmt.Printf("Freeblock bitmap %v\n", fs.freeBlockBitMap)
	return nil
}

func (fs *FS) Format(disk *disk.Disk) bool {
	var err error
	var sblock SuperBlock = SuperBlock{
		MagicNumber: MAGIC_NUMBER,
		Blocks:      uint32(disk.Blocks),
		InodeBlocks: uint32(math.Round(float64(disk.Blocks) * 0.1)),
		Inodes:      0,
	}
	buf := bytes.NewBuffer(make([]byte, 0))
	// Write superblock
	err = binary.Write(buf, enc, &sblock)
	if err != nil {
		fmt.Println(fmt.Errorf("could not format: %s", err.Error()))
		return false
	}

	err = disk.Write(0, buf.Bytes())
	if err != nil {
		fmt.Println(fmt.Errorf("could not format: %s", err.Error()))
		return false
	}

	// clear all inode blocks
	err = fs.clearInodeBlocks(disk, &sblock, buf)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	// clear all data blocks
	err = fs.clearDataBlocks(disk, &sblock, buf)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

func (fs *FS) Mount(disk *disk.Disk) bool {
	// Read superblock
	err := fs.loadSuperBlock(disk, &fs.superBlock)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to mount disk: %s", err.Error()))
		return false
	}
	// copy Inode blocks
	fs.inodeBlocks = make([]*InodeBlock, fs.superBlock.InodeBlocks)
	err = fs.loadInodeBlocks(disk, int(fs.superBlock.InodeBlocks), fs.inodeBlocks)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to mount disk: %s", err.Error()))
		return false
	}

	// initialize free block bitmap
	err = fs.initFreeBlockBitMap(disk)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to mount disk: %s", err.Error()))
		return false
	}
	disk.Mount()
	fs.disk = disk
	return true
}

func (fs *FS) Read(inumber int) (data DataBlock, err error) {
	var buf [disk.BLOCK_SIZE]byte
	data = DataBlock{}
	// read inode block from disk
	err = fs.disk.Read(inumber, buf[:])
	if err != nil {
		return
	}
	err = binary.Read(bytes.NewBuffer(buf[:]), enc, &data)
	return
}

func (fs *FS) Write(inumber int, data DataBlock) error {
	if !fs.isfreeblock(inumber) {
		return fmt.Errorf("trying to write to a used inode block")
	}
	buf := &bytes.Buffer{}
	// write disk to inode block
	err := binary.Write(buf, enc, &data)
	if err != nil {
		return err
	}
	err = fs.disk.Write(inumber, buf.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to write to inode block: %s", err.Error())
	}
	// update free block bitmap
	fs.freeBlockBitMap[inumber] = 1
	return nil 
}

func (fs *FS) Remove(inumber int) error {
	dblock := DataBlock{}
	buf := &bytes.Buffer{}
	err := binary.Write(buf, enc, &dblock)
	// clear inode block
	err = fs.disk.Write(inumber, buf.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to clear inode block: %s", err.Error())
	}
	// update free block bitmap
	fs.freeBlockBitMap[inumber] = 0
	return nil
}

func (fs *FS) Stat(inumber int) (int, error) {
	dblock, err := fs.Read(inumber)
	if err != nil {
		return -1, fmt.Errorf("Could not read inode block: %s", err.Error())
	}
	return len(dblock.Data), nil 
}


func (fs *FS) Create() (inumber int, err error) {
	inumber = -1
	iblocks := make([]*InodeBlock, fs.superBlock.InodeBlocks)
	// Read Inode blocks
	err = fs.loadInodeBlocks(fs.disk, int(fs.superBlock.InodeBlocks), iblocks)
	if err != nil {
		return 
	}

	for _, iblock := range iblocks {
		for id, inode := range iblock.Inodes {
			if inode.Valid == 0 {
				inumber = id 
				inode.Valid = 1
				// set inode block index as used
				fs.freeBlockBitMap[inumber] = 1
				return 
			}
		}

	}
	return
}

/* utillity filesystem functions */

func (fs *FS) initFreeBlockBitMap(dsk *disk.Disk) error {
	var Pointers [POINTERS_PER_BLOCK]uint32
	var buf [disk.BLOCK_SIZE]byte = [disk.BLOCK_SIZE]byte{}
	fs.freeBlockBitMap = make([]uint32, fs.superBlock.Blocks)
	// set super block (0) as used
	fs.freeBlockBitMap[0] = 1
	for _, iblock := range fs.inodeBlocks {
		for _, inode := range iblock.Inodes {
			// TODO: also check that inode is valid i.e inode.Valid?
			if inode.Size > 0 {
				for _, dblock := range inode.Direct {
					fs.freeBlockBitMap[dblock] = 1
				}
				if inode.Indirect > 0 {
					fs.freeBlockBitMap[inode.Indirect] = 1
					err := dsk.Read(int(inode.Indirect), buf[:])
					if err != nil {
						return fmt.Errorf("failed to return indirect block (%d): %s", inode.Indirect, err.Error())
					}
					err = binary.Write(bytes.NewBuffer(buf[:]), enc, Pointers)
					if err != nil {
						return fmt.Errorf("failed to return indirect block  pointers(%d): %s", inode.Indirect, err.Error())
					}
					fmt.Printf("Indirect block pointers: %v", Pointers)
					for _, p := range Pointers {
						fs.freeBlockBitMap[p] = 1
					}
				}

			}
		}

	}
	return nil
}

func (fs *FS) clearInodeBlocks(dsk *disk.Disk, sblock *SuperBlock, buf *bytes.Buffer) error {
	var err error
	var iblock InodeBlock = InodeBlock{}
	for i := uint32(1); i <= sblock.InodeBlocks; i++ {
		buf.Reset()
		err = binary.Write(buf, enc, &iblock)
		if err != nil {
			return fmt.Errorf("could not format: %s", err.Error())
		}
		err = dsk.Write(int(i), buf.Bytes())
		if err != nil {
			return fmt.Errorf("could not format: %s", err.Error())
		}
	}

	return nil
}

func (fs *FS) clearDataBlocks(dsk *disk.Disk, sblock *SuperBlock, buf *bytes.Buffer) error {
	var dblock DataBlock = DataBlock{}
	var err error
	for i := sblock.InodeBlocks + 1; i < sblock.Blocks; i++ {
		buf.Reset()
		err = binary.Write(buf, enc, &dblock)
		if err != nil {
			return fmt.Errorf("could not format: %s", err.Error())
		}
		err = dsk.Write(int(i), buf.Bytes())
		if err != nil {
			return fmt.Errorf("could not format: %s", err.Error())
		}
	}
	return nil
}

func (fs *FS) loadInodeBlock(dsk *disk.Disk, blocknum int, block *InodeBlock) error {
	var buf [disk.BLOCK_SIZE]byte
	err := dsk.Read(blocknum, buf[:])
	err = binary.Read(bytes.NewBuffer(buf[:]), enc, block)
	if err != nil {
		return fmt.Errorf("failed to read inode block: %s", err.Error())
	}
	return nil
}

func (fs *FS) loadInodeBlocks(dsk *disk.Disk, inodeblocks int, block []*InodeBlock) error {
	var err error
	for i := 1; i <= inodeblocks; i++ {
		block[i-1] = &InodeBlock{}
		err = fs.loadInodeBlock(dsk, i, block[i-1])
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *FS) loadSuperBlock(dsk *disk.Disk, sblock *SuperBlock) error {
	var buf [disk.BLOCK_SIZE]byte
	err := dsk.Read(0, buf[:])
	if err != nil {
		return err
	}
	err = binary.Read(bytes.NewBuffer(buf[:]), enc, sblock)
	if err != nil {
		return fmt.Errorf("failed to read superblock: %s", err.Error())
	}
	return nil
}

func (fs *FS) isfreeblock(inumber int ) bool {
	if len(fs.freeBlockBitMap) < inumber {
		return false
	}
	if fs.freeBlockBitMap[inumber] != 0 {
		return false
	}
	return true
}