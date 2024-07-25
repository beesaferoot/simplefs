package fs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"simplefs/internal/disk"
	"strconv"
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

	Write(int, []byte) (*Inode, error)
	Read(int) (*Inode, error)
	Cat(int) error
}

type FS struct {
	disk            *disk.Disk
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
	if sblock.MagicNumber == MAGIC_NUMBER {
		fmt.Printf("    magic number is valid\n")
	}
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
				fmt.Printf("    direct blocks: %s\n", mapToString(v.Direct[:]))
				if v.Indirect > 0 {
					fmt.Printf("    indirect blocks: %v\n", v.Indirect)
				}

			}
		}

	}
	fmt.Printf("%d disk block reads\n", dsk.Reads)
	fmt.Printf("%d disk block writes\n", dsk.Writes)
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

func (fs *FS) Read(inumber int) (inode *Inode, err error) {
	inode, err = fs.loadInode(inumber)
	if err != nil {
		return nil, fmt.Errorf("failed to read inode %d: %s", inumber, err.Error())
	}
	return
}

func (fs *FS) ReadDataBlock(blocknum int) (data DataBlock, err error) {
	var buf [disk.BLOCK_SIZE]byte
	data = DataBlock{}
	// read inode block from disk
	err = fs.disk.Read(blocknum, buf[:])
	if err != nil {
		return
	}
	err = binary.Read(bytes.NewBuffer(buf[:]), enc, &data)
	return
}

func (fs *FS) loadInode(inumber int) (inode *Inode, err error) {
	iblocks := make([]*InodeBlock, fs.superBlock.InodeBlocks)
	// Read Inode blocks
	err = fs.loadInodeBlocks(fs.disk, int(fs.superBlock.InodeBlocks), iblocks)
	if err != nil {
		return
	}
	for _, iblock := range iblocks {
		for id, v := range iblock.Inodes {
			if id == inumber {
				return &v, nil
			}
		}
	}
	return
}

func (fs *FS) Write(inumber int, data []byte) (inode *Inode, err error) {

	iblocks := make([]*InodeBlock, fs.superBlock.InodeBlocks)
	// Read Inode blocks
	err = fs.loadInodeBlocks(fs.disk, int(fs.superBlock.InodeBlocks), iblocks)

	if err != nil {
		return
	}

	var inodeBlockIdx int
	var inodeBlock *InodeBlock
	for idx, iblock := range iblocks {
		for id, v := range iblock.Inodes {
			if id == inumber {
				inode = &v
				inodeBlockIdx = idx + 1
				inodeBlock = iblock
				break
			}
		}
	}

	if inode == nil {
		return nil, fmt.Errorf("failed to write inode %d: inode not found", inumber)
	}

	bytesToWrite := len(data)

	blocknum := int(fs.superBlock.InodeBlocks) + 1

	// search for free disk blocks to write inode data into
	for {
		if !fs.isValidBlock(blocknum) {
			return nil, errors.New("attempt to access disk block")
		}

		if fs.isFreeblock(blocknum) {
			break
		}
		blocknum += 1
	}
	err = fs.disk.Write(blocknum, data)

	if err != nil {
		return nil, fmt.Errorf("failed to write to inode block: %s", err.Error())
	}

	inode.Size += uint32(bytesToWrite)

	for idx, bn := range inode.Direct {
		if bn == 0 {
			inode.Direct[idx] = uint32(blocknum)
		}
	}

	(*inodeBlock).Inodes[inumber] = *inode

	var buf [disk.BLOCK_SIZE]byte
	writeBuf := bytes.NewBuffer(buf[:0])
	err = binary.Write(writeBuf, enc, inodeBlock)

	if err != nil {
		return nil, fmt.Errorf("failed to write to inode block: %s", err.Error())
	}

	err = fs.disk.Write(inodeBlockIdx, writeBuf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("failed to write to inode block: %s", err.Error())
	}

	// update free block bitmap
	fs.freeBlockBitMap[blocknum] = 1

	return inode, nil
}

func (fs *FS) Remove(inumber int) error {
	errMsg := "failed to remove to data block (%d): %s"

	iblocks := make([]*InodeBlock, fs.superBlock.InodeBlocks)
	// Read Inode blocks
	err := fs.loadInodeBlocks(fs.disk, int(fs.superBlock.InodeBlocks), iblocks)

	if err != nil {
		return err
	}

	var inode Inode
	var inodeBlockIdx int
	var inodeBlock *InodeBlock
	for idx, iblock := range iblocks {
		for id, v := range iblock.Inodes {
			if id == inumber {
				inode = v
				inodeBlockIdx = idx + 1
				inodeBlock = iblock
				break
			}
		}
	}

	var buf [disk.BLOCK_SIZE]byte

	for idx, blocknum := range inode.Direct {
		if blocknum > 0 {
			err = fs.disk.Write(int(blocknum), buf[:])
			if err != nil {
				return fmt.Errorf(errMsg, blocknum, err.Error())
			}
			inode.Direct[idx] = 0
			fs.freeBlockBitMap[blocknum] = 0
		}

	}

	if inode.Indirect > 0 {
		blocknum := int(inode.Indirect)
		err = fs.disk.Write(int(inode.Indirect), buf[:])
		if err != nil {
			return fmt.Errorf(errMsg, blocknum, err.Error())
		}
		fs.freeBlockBitMap[blocknum] = 0
	}

	// reset inode
	inode.Valid = 0
	inode.Size = 0
	inode.Indirect = 0

	// write update inode back into disk
	(*inodeBlock).Inodes[inumber] = inode
	writeBuf := bytes.NewBuffer(buf[:0])
	err = binary.Write(writeBuf, enc, inodeBlock)

	if err != nil {
		return fmt.Errorf(errMsg, inumber, err.Error())
	}

	err = fs.disk.Write(inodeBlockIdx, writeBuf.Bytes())

	if err != nil {
		return fmt.Errorf(errMsg, inumber, err.Error())
	}

	return nil
}

func (fs *FS) Stat(inumber int) (int, error) {
	inode, err := fs.Read(inumber)
	if err != nil {
		return -1, fmt.Errorf("could not read inode block: %s", err.Error())
	}
	return int(inode.Size), nil
}

func (fs *FS) Create() (inumber int, err error) {
	inumber = -1
	iblocks := make([]*InodeBlock, fs.superBlock.InodeBlocks)
	// Read Inode blocks
	err = fs.loadInodeBlocks(fs.disk, int(fs.superBlock.InodeBlocks), iblocks)
	if err != nil {
		return
	}

	var inode Inode
	var inodeBlockIdx int
	var inodeBlock *InodeBlock

	for idx, iblock := range iblocks {
		for id, v := range iblock.Inodes {
			if id > 0 && v.Valid == 0 {
				inumber = id
				inode = v
				inodeBlockIdx = idx + 1
				inodeBlock = iblock
				break
			}
		}

	}

	var buf [disk.BLOCK_SIZE]byte

	inode.Valid = 1

	(*inodeBlock).Inodes[inumber] = inode
	writeBuf := bytes.NewBuffer(buf[:0])
	err = binary.Write(writeBuf, enc, inodeBlock)

	if err != nil {
		return inumber, fmt.Errorf("failed to write to inode block: %s", err.Error())
	}

	err = fs.disk.Write(inodeBlockIdx, writeBuf.Bytes())

	return
}

func (fs *FS) Cat(inumber int) error {
	inode, err := fs.Read(inumber)
	if err != nil {
		return fmt.Errorf("could not read inode block: %s", err.Error())
	}

	for _, dblockNum := range inode.Direct {
		if dblockNum > 0 {
			dblock, err := fs.ReadDataBlock(int(dblockNum))
			if err != nil {
				return fmt.Errorf("could not read inode block: %s", err.Error())
			}
			fmt.Printf("%s", dblock.Data[:])
		}

	}

	return nil
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
			if inode.Size > 0 && inode.Valid == 1 {
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
						if p != 0 {
							fs.freeBlockBitMap[p] = 1
						}

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

func (fs *FS) isValidBlock(blocknum int) bool {
	return len(fs.freeBlockBitMap) > blocknum
}

func (fs *FS) isFreeblock(blocknum int) bool {
	return fs.freeBlockBitMap[blocknum] == 0
}

func mapToString(arr []uint32) string {
	res := ""

	for v := range arr {
		res += strconv.Itoa(int(arr[v])) + " "
	}

	return res
}
