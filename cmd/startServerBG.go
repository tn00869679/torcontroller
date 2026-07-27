package cmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Seicrypto/torcontroller/internal/controller"
	"github.com/spf13/cobra"
)

var StartBackgroundCmd = &cobra.Command{
	Use:   "start-background",
	Short: "Start Torcontroller listener as a background process",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info(fmt.Sprintf("Listener started successfully at %s.\n", socketPath))
		os.Remove(socketPath)
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			log.Error(err.Error())
			return
		}
		defer func() {
			listener.Close()
			os.Remove(socketPath)
		}()
		os.Chmod(socketPath, 0777)

		// Loop for accepting connections
		for {
			log.Info("Waiting for connection...")
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Warn("Client closed the connection.")
					continue
				}
				log.Error(fmt.Sprintf("Error accepting connection: %v", err))
				continue
			}
			log.Info("Connection established")

			// go controller.HandleConnection(conn, socketPath, listener)
			go func(c net.Conn) {
				if err := controller.HandleConnection(c, socketPath, listener); err != nil {
					log.Error(fmt.Sprintf("Error handling connection: %v", err))
				}
			}(conn)
		}
	},
}
