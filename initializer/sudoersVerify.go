package initializer

import (
	"fmt"
	"os"
	"syscall"
)

func (i *Initializer) SudoersFileVerify() bool {
	sudoersPath := "/etc/sudoers.d/torcontroller"

	fileInfo, err := i.FileSystem.Stat(sudoersPath)
	if os.IsNotExist(err) {
		fmt.Println("- Sudoers configuration [MISSING]")
		return false
	} else if err != nil {
		fmt.Printf("Failed to check sudoers file: %v\n", err)
		return false
	}

	if fileInfo.Mode() != 0o440 {
		fmt.Println("- Sudoers configuration [INVALID PERMISSIONS]")
		return false
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || stat.Gid != 0 {
		fmt.Println("- Sudoers configuration [INVALID OWNER]")
		return false
	}

	cmd := []string{"sudo", "visudo", "-cf", sudoersPath}
	if _, err := i.CommandRunner.Run(cmd[0], cmd[1:]...); err != nil {
		fmt.Println("- Sudoers configuration [INVALID SYNTAX]")
		return false
	}

	return true
}
