package shell

import (
	"bufio"
	"fmt"
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

		case "quit", "exit":
			return
		default:
			shell.helpCmd()
			break
		}
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
