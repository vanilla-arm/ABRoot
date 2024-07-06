package core

/*	License: GPLv3
	Authors:
		Mirko Brombin <mirko@fabricators.ltd>
		Vanilla OS Contributors <https://github.com/vanilla-os/>
	Copyright: 2024
	Description:
		ABRoot is utility which provides full immutability and
		atomicity to a Linux system, by transacting between
		two root filesystems. Updates are performed using OCI
		images, to ensure that the system is always in a
		consistent state.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var abrootDir = "/etc/abroot"

func init() {
	if !RootCheck(false) {
		return
	}

	if _, err := os.Stat(abrootDir); os.IsNotExist(err) {
		err := os.Mkdir(abrootDir, 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func RootCheck(display bool) bool {
	if os.Geteuid() != 0 {
		if display {
			fmt.Println("You must be root to run this command")
		}

		return false
	}

	return true
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		PrintVerboseInfo("fileExists", "File exists:", path)
		return true
	}

	PrintVerboseInfo("fileExists", "File does not exist:", path)
	return false
}

// isLink checks if a path is a link
func isLink(path string) bool {
	if _, err := os.Lstat(path); err == nil {
		PrintVerboseInfo("isLink", "Path is a link:", path)
		return true
	}

	PrintVerboseInfo("isLink", "Path is not a link:", path)
	return false
}

// CopyFile copies a file from source to dest
func CopyFile(source, dest string) error {
	PrintVerboseInfo("CopyFile", "Running...")

	PrintVerboseInfo("CopyFile", "Opening source file")
	srcFile, err := os.Open(source)
	if err != nil {
		PrintVerboseErr("CopyFile", 0, err)
		return err
	}
	defer srcFile.Close()

	PrintVerboseInfo("CopyFile", "Opening destination file")
	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		PrintVerboseErr("CopyFile", 1, err)
		return err
	}
	defer destFile.Close()

	PrintVerboseInfo("CopyFile", "Performing copy operation")
	if _, err := io.Copy(destFile, srcFile); err != nil {
		PrintVerboseErr("CopyFile", 2, err)
		return err
	}

	return nil
}

// isDeviceLUKSEncrypted checks whether a device specified by devicePath is a LUKS-encrypted device
func isDeviceLUKSEncrypted(devicePath string) (bool, error) {
	PrintVerboseInfo("isDeviceLUKSEncrypted", "Verifying if", devicePath, "is encrypted")

	isLuksCmd := "cryptsetup isLuks %s"

	cmd := exec.Command("sh", "-c", fmt.Sprintf(isLuksCmd, devicePath))
	err := cmd.Run()
	if err != nil {
		// We expect the command to return exit status 1 if partition isn't
		// LUKS-encrypted
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return false, nil
			}
		}
		err = fmt.Errorf("failed to check if %s is LUKS-encrypted: %s", devicePath, err)
		PrintVerboseErr("isDeviceLUKSEncrypted", 0, err)
		return false, err
	}

	return true, nil
}

// getDirSize calculates the total size of a directory recursively.
// FIXME: find a way to avoid using du and any other external command
//
//	the walk function in the os package can be used to calculate the size
//	of a directory but it needs a solid implementation for links
func getDirSize(path string) (int64, error) {
	cmd := exec.Command("du", "-s", "-b", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	output := strings.Fields(out.String())
	if len(output) == 0 {
		return 0, errors.New("failed to get directory size")
	}

	size, err := strconv.ParseInt(output[0], 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}
