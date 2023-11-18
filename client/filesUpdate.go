// updatefiles.go

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var prevFileList []string // To store the previous file list

func watchForFileUpdates(conn net.Conn, directory string) {
	fileTicker := time.NewTicker(1 * time.Second) // Check for updates every 1 seconds
	defer fileTicker.Stop()

	prevFileList, err := listFiles(directory)
	if err != nil {
		fmt.Println("Error listing files in the directory:", err)
		return
	}

	for range fileTicker.C {
		fileList, err := listFiles(directory)
		if err != nil {
			fmt.Println("Error listing files in the directory:", err)
			continue
		}

		// Compare the current file list with the previous one
		filesUpdated := false
		if len(fileList) != len(prevFileList) {
			filesUpdated = true
		} else {
			for i, file := range fileList {
				if file != prevFileList[i] {
					filesUpdated = true
					break
				}
			}
		}

		if filesUpdated {
			// Files have been updated, send the updated file list to the server
			sendFileUpdate(conn, fileList)

		}

		// Update the previous file list for the next iteration
		prevFileList = fileList
	}
}

func sendFileUpdate(conn net.Conn, fileList []string) {
	command := "UPDATE"
	message := command + " " + strings.Join(fileList, " ")

	_, err := conn.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println("Error sending file update to the server:", err)
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Print(response)
}

func listFiles(directory string) ([]string, error) {
	var fileList []string
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileList = append(fileList, file.Name())
	}

	return fileList, nil
}
