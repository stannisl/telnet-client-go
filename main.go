package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Парсинг аргументов командной строки
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: telnet-client [--timeout duration] host port")
		os.Exit(1)
	}

	host := args[0]
	port := args[1]

	// Создание клиента
	client := NewClient(host, port, *timeout)

	// Запуск клиента
	if err := client.Run(context.Background()); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}

	fmt.Println("Telnet client finished successfully")
}
