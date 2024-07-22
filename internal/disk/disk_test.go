package disk

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiskEmulator(t *testing.T){
	for scenario, fn := range map[string] func (t *testing.T, d *Disk, tfn func(*Disk)){
		"open disk image": testOpen,
		"write to disk": testWrite,
		"read from disk": testRead,
	}{
		t.Run(scenario, func(t *testing.T) {
			disk := &Disk{}
			fn(t, disk, tearDown)
		})
	}
}

func tearDown(disk *Disk){
	err := disk.Close()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(disk.Name); err != nil {
		log.Fatal(err)
	}
}

func testOpen(t *testing.T, d *Disk, tfn func(*Disk)){
	defer tfn(d)
	err := d.Open("test_image10", 10)
	require.NoError(t, err)
	require.Equal(t, 10, int(d.Blocks))
	require.Equal(t, "test_image10", d.Name)
	require.NotEqual(t, 0, d.FileDescriptor)
	require.Equal(t, 0, int(d.Reads))
	
	err = d.Open("test_image30", 30)
	require.NoError(t, err)
	require.Equal(t, 30, int(d.Blocks))
	require.Equal(t, "test_image30", d.Name)
	require.NotEqual(t, 0, d.FileDescriptor)
	require.Equal(t, 0, int(d.Reads))
}

func testWrite(t *testing.T, d *Disk, tfn func(*Disk)){
	defer tfn(d)
	err := d.Open("test_image10", 10)
	require.NoError(t, err)
	data := make([]byte, BLOCK_SIZE)
	copy(data, "hello world!!")
	err = d.Write(0, data)
	require.NoError(t, err)
	require.Equal(t, 1, int(d.Writes))
	
	err = d.Write(1, data)
	require.NoError(t, err)
	require.Equal(t, 2, int(d.Writes))
	
	// write to the same block
	err = d.Write(1, data)
	require.NoError(t, err)
	require.Equal(t, 3, int(d.Writes))

}

func testRead(t *testing.T, d *Disk, tfn func(*Disk)){
	defer tfn(d)
	err := d.Open("test_image10", 10)
	require.NoError(t, err)
	wdata := make([]byte, BLOCK_SIZE)
	copy(wdata, "hello world!!")
	err = d.Write(0, wdata)
	require.NoError(t, err)
	// read from written block (0)
	rdata := make([]byte, BLOCK_SIZE)
	err = d.Read(0, rdata)
	require.NoError(t, err)
	require.Equal(t, 1, int(d.Reads))
	require.Equal(t, "hello world!!", string(rdata[:13]))

	// read from empty block 
	err = d.Read(1, rdata)
	require.NoError(t, err)
	require.Equal(t, 2, int(d.Reads))
	// read null bytes ("\x00")
	require.NotEqual(t, "hello world!!", string(rdata[:13]))
}