package fs

import (
	"log"
	"os"
	"simplefs/internal/disk"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFsDebug(t *testing.T) {
	var fs FileSystem = NewFS()
	disk := &disk.Disk{}
	defer disk.Close()
	err := disk.Open("../../data/image.20", 20)
	require.NoError(t, err)
	err = fs.Debug(disk)
	require.NoError(t, err)
}

func TestFsMount(t *testing.T) {
	var fs FileSystem = NewFS()
	disk := &disk.Disk{}
	defer disk.Close()
	err := disk.Open("../../data/image.200", 200)
	require.NoError(t, err)
	ok := fs.Mount(disk)
	require.Equal(t, true, ok)
	err = fs.Debug(disk)
	require.NoError(t, err)
}

func TestFsFormat(t *testing.T) {
	var fs FileSystem = NewFS()
	disk := &disk.Disk{}
	defer tearDown(disk)
	err := disk.Open("test-image.10", 10)
	require.NoError(t, err)
	ok := fs.Format(disk)
	require.Equal(t, true, ok)
	err = fs.Debug(disk)
	require.NoError(t, err)
}

func TestFsRead(t *testing.T) {
	var fs = NewFS()
	disk := &disk.Disk{}
	defer disk.Close()
	err := disk.Open("../../data/image.5", 5)
	fs.Debug(disk)
	require.NoError(t, err)
	ok := fs.Mount(disk)
	require.Equal(t, true, ok)
	inode, err := fs.Read(1)
	require.NoError(t, err)
	require.Equal(t, inode.Size > 0, true)
	require.Equal(t, inode.Valid == 1, true)
}

func TestFsWrite(t *testing.T) {
	var fs = NewFS()
	disk := &disk.Disk{}
	err := disk.Open("../../data/image-test.5", 5)
	require.NoError(t, err)
	ok := fs.Mount(disk)
	require.Equal(t, true, ok)
	ok = fs.Format(disk)
	require.Equal(t, true, ok)
	data := []byte("hello world")
	// create inode
	inodeNum, err := fs.Create()
	require.NoError(t, err)
	inode, err := fs.Write(inodeNum, data)

	require.NoError(t, err)
	require.Equal(t, inode.Size > 0, true)
	require.Equal(t, inode.Valid == 1, true)
	require.Equal(t, inode.Direct[0] > 0, true)

}

func TestFsRemove(t *testing.T) {
	var fs = NewFS()
	disk := &disk.Disk{}
	err := disk.Open("../../data/image-test.5", 5)
	require.NoError(t, err)
	ok := fs.Mount(disk)
	require.Equal(t, true, ok)

	ok = fs.Format(disk)
	require.Equal(t, true, ok)

	// create inode
	inodeNum, err := fs.Create()
	require.NoError(t, err)

	err = fs.Remove(inodeNum)

	require.NoError(t, err)

	// trying to remove a non-existing inode
	err = fs.Remove(inodeNum)
	require.NoError(t, err)
}

func TestFsStat(t *testing.T) {
	// inode 1 (965) bytes
	var fs = NewFS()
	disk := &disk.Disk{}
	err := disk.Open("../../data/image.5", 5)
	require.NoError(t, err)
	ok := fs.Mount(disk)
	require.Equal(t, true, ok)

	inode := 1
	size, err := fs.Stat(inode)

	require.NoError(t, err)
	require.Equal(t, 965, size)
}

func tearDown(disk *disk.Disk) {
	err := disk.Close()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(disk.Name); err != nil {
		log.Fatal(err)
	}
}
