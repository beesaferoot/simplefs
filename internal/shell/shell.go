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
		fmt.Printf("Failed to open disk: %s\n", err)
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
			fmt.Printf("Failed to read input: %s\n", err)
			return
		}

		input = strings.TrimSpace(input)

		args := strings.Split(input, " ")

		switch args[0] {
		case "help":
			shell.helpCmd()
			break
		case "format":
			shell.filesystem.Format(shell.disk)
			break
		case "mount":
			shell.filesystem.Mount(shell.disk)
			break
		case "debug":
			err := shell.filesystem.Debug(shell.disk)
			if err != nil {
				fmt.Printf("Failure on debug command: %s\n", err)
			}
			break
		case "stat":
			if len(args) < 2 {
				fmt.Printf("Usage: stat <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				shell.filesystem.Stat(inode)
			}
			break
		case "cat":
			if len(args) < 2 {
				fmt.Printf("Usage: cat <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				shell.filesystem.Cat(inode)
			}
			break
		case "remove":
			if len(args) < 2 {
				fmt.Printf("Usage: remove <inode>\n")
			} else {
				inode, _ := strconv.Atoi(args[1])
				err := shell.filesystem.Remove(inode)
				if err != nil {
					fmt.Printf("Failure on remove command: %s\n", err)
				}
			}
			break
		case "exit":
		case "quit":
			return
		default:
			break
		}
	}
}

func (shell *Shell) helpCmd() {
	fmt.Println(`Commands are:
	format
	mount
	debug
	remove  <inode>
	cat     <inode>
	stat    <inode>
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
