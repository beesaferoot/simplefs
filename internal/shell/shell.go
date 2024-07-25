package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	ds "simplefs/internal/disk"
	"simplefs/internal/fs"
	"strconv"
	"strings"
)

type Shell struct {
	filesystem fs.FileSystem
	disk       *ds.Disk
}

func NewShell(path string, nblocks int) *Shell {
	disk := &ds.Disk{}
	err := disk.Open(path, nblocks)
	if err != nil {
		fmt.Printf("failed to open disk: %s\n", err.Error())
		os.Exit(1)
	}
	filesystem := fs.NewFS()
	return &Shell{
		filesystem: filesystem,
		disk:       disk,
	}
}

func (shell *Shell) Init() {
	defer shell.Shutdown()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("sfs> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("failed to read input: %s\n", err.Error())
			return
		}

		input = strings.TrimSpace(input)

		args := strings.Split(input, " ")

		switch args[0] {
		case "help":
			shell.helpCmd()
			break
		case "format":
			ok := shell.filesystem.Format(shell.disk)
			if ok {
				fmt.Println("disk formatted.")
			}
			break
		case "mount":
			ok := shell.filesystem.Mount(shell.disk)
			if ok {
				fmt.Println("disk mounted.")
			}
			break
		case "debug":
			err := shell.filesystem.Debug(shell.disk)
			if err != nil {
				fmt.Printf("failure on debug command: %s\n", err.Error())
			}
			break
		case "stat":
			if len(args) < 2 {
				fmt.Printf("Usage: stat <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				numBytes, err := shell.filesystem.Stat(inode)
				if err != nil {
					fmt.Printf("failure on stat command: %s\n", err.Error())
				} else {
					fmt.Printf("inode %d has size %d bytes.\n", inode, numBytes)
				}
			}
			break
		case "cat":
			if len(args) < 2 {
				fmt.Printf("Usage: cat <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				err := shell.filesystem.Cat(inode)
				if err != nil {
					fmt.Printf("failure on cat command: %s\n", err.Error())
				}
			}
			break
		case "remove":
			if len(args) < 2 {
				fmt.Printf("Usage: remove <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				err := shell.filesystem.Remove(inode)
				if err != nil {
					fmt.Printf("failure on remove command: %s\n", err.Error())
				} else {
					fmt.Printf("removed inode %d.\n", inode)
				}
			}
			break
		case "create":
			inode, err := shell.filesystem.Create()
			if err != nil {
				fmt.Printf("failure on create command: %s\n", err.Error())
			} else {
				fmt.Printf("created inode %d.\n", inode)
			}
		case "copyout":
			if len(args) < 3 {
				fmt.Printf("Usage: copyout <inode> <file>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				filePath := args[2]
				bytesCopied, err := shell.CopyOut(inode, filePath)
				if err != nil {
					fmt.Printf("failed on copyout command: %s\n", err.Error())
					return
				}
				fmt.Printf("%d bytes copied\n", bytesCopied)
			}
			break
		case "copyin":
			if len(args) < 3 {
				fmt.Printf("Usage: copyin <file> <inode>\n")
			} else {
				filePath := args[1]
				inode, _ := strconv.Atoi(args[2])

				bytesCopied, err := shell.CopyIn(filePath, inode)
				if err != nil {
					fmt.Printf("failed on copyin command: %s\n", err.Error())
					return
				}
				fmt.Printf("%d bytes copied\n", bytesCopied)
			}
			break
		case "quit", "exit":
			return
		default:
			shell.helpCmd()
			break
		}
	}
}

func (shell *Shell) CopyOut(inumber int, filePath string) (int32, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE, 0600)
	defer file.Close()

	if err != nil {
		return -1, err
	}

	inode, err := shell.filesystem.Read(inumber)

	if err != nil {
		return -1, err
	}

	writeBuf := bufio.NewWriter(file)

	var readBuf [ds.BLOCK_SIZE]byte

	bytesCopied := 0

	for _, blocknum := range inode.Direct {
		if blocknum > 0 {
			err := shell.disk.Read(int(blocknum), readBuf[:])
			if err != nil {
				return -1, err
			}
			n, err := writeBuf.Write(readBuf[:])
			if err != nil {
				return -1, err
			}
			bytesCopied += n
		}

	}

	return int32(bytesCopied), nil

}

func (shell *Shell) CopyIn(filePath string, inumber int) (int32, error) {
	file, err := os.Open(filePath)
	defer file.Close()

	if err != nil {
		return -1, err
	}

	readBuf := bufio.NewReader(file)
	var writeBuf [ds.BLOCK_SIZE]byte
	bytesCopied := 0

	for {
		n, err := readBuf.Read(writeBuf[:])

		if err == io.EOF {
			return int32(bytesCopied), nil
		}

		if err != nil {
			return -1, err
		}

		_, err = shell.filesystem.Write(inumber, writeBuf[:n])

		if err != nil {
			return -1, err
		}

		bytesCopied += n

	}

}

func (shell *Shell) helpCmd() {
	fmt.Println(`Commands are:
	format
	mount
	debug
	create
	remove  <inode>
	cat     <inode>
	stat    <inode>
	copyin  <file> <inode>
	copyout <inode> <file>
	help
	quit
	exit`)
}

func (shell *Shell) Shutdown() {
	err := shell.disk.Close()

	if err != nil {
		fmt.Printf("Failed to close disk: %s\n", err)
	}
}
