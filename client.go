package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Client struct {
	host    string
	port    string
	timeout time.Duration
}

func NewClient(host, port string, timeout time.Duration) *Client {
	return &Client{
		host:    host,
		port:    port,
		timeout: timeout,
	}
}

func (c *Client) Run(ctx context.Context) error {
	// Установка соединения с таймаутом
	address := net.JoinHostPort(c.host, c.port)
	conn, err := net.DialTimeout("tcp", address, c.timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", address)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Группа для управления горутинами
	g, ctx := errgroup.WithContext(ctx)

	// Горутина для чтения из сокета и вывода в STDOUT
	g.Go(func() error {
		return c.readFromSocket(ctx, conn)
	})

	// Горутина для чтения из STDIN и записи в сокет
	g.Go(func() error {
		return c.writeToSocket(ctx, conn)
	})

	// Обработка сигналов завершения
	g.Go(func() error {
		return c.handleSignals(ctx, cancel)
	})

	// Ожидаем завершения всех горутин
	return g.Wait()
}

func (c *Client) readFromSocket(ctx context.Context, conn net.Conn) error {
	reader := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Устанавливаем таймаут на чтение
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					fmt.Println("Connection closed by server")
					return err
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Таймаут - продолжаем цикл
					continue
				}
				return err
			}

			fmt.Print(line)
		}
	}
}

func (c *Client) writeToSocket(ctx context.Context, conn net.Conn) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					fmt.Println("\nCtrl+D pressed - closing connection")
					return err
				}
				return err
			}

			_, err = conn.Write([]byte(line))
			if err != nil {
				return fmt.Errorf("write to socket failed: %w", err)
			}
		}
	}
}

func (c *Client) handleSignals(ctx context.Context, cancel context.CancelFunc) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		return nil
	case sig := <-sigCh:
		fmt.Printf("\nReceived signal: %v. Closing connection...\n", sig)
		cancel()
		return fmt.Errorf("received signal: %v", sig)
	}
}
