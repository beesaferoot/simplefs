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
	//var fs FileSystem = NewFS()
	//disk := &disk.Disk{}
	//defer disk.Close()
	//err := disk.Open("../../data/image.5", 5)
	//fs.Debug(disk)
	//require.NoError(t, err)
	//ok := fs.Mount(disk)
	//require.Equal(t, true, ok)
	//dblock, err := fs.Read(0)
	//require.NoError(t, err)
	//require.Equal(t, len(dblock.Data) > 0, true)
	//log.Printf("%v", dblock.Data)
}

func TestFsWrite(t *testing.T) {

}

func TestFsRemove(t *testing.T) {

}

func TestFsStat(t *testing.T) {

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
