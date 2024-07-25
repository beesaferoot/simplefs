## Simplefs

A basic unix-like filesystem using a disk emulator. This implementation is based on a Notre Dame CS course filesystem [project](https://www3.nd.edu/~pbui/teaching/cse.30341.fa17/project06.html)

## Usage 


To access the filesystem you need a shell, specify a byte data file used by the disk emulator i.e 
```bash
$ go build .
$ ./simplefs image.5 5
```

use the help command to see available filesystem commands.
```
sfs> help
Commands are:
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
        exit
sfs> 
```

### Testing

Run command to execute unit tests.
```bash
$ make test
```

### License

simplefs is distributed under the [MIT License.](https://github.com/beesaferoot/simplefs/blob/main/LICENSE.txt)