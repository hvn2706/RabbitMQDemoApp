package main

import (
	"fmt"
	"io/ioutil"
	"log"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var ch *amqp.Channel
var queue amqp.Queue

// Main Function

func main() {
	init_connection()
	defer close_connection()
	start_server()
}

// RabbitMQ Functions

func init_connection() {
	var err error
	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	ch, err = conn.Channel()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	queue, err = ch.QueueDeclare(
		"test_queue", // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println("Connected to RabbitMQ")
}

func close_connection() {
	ch.Close()
	conn.Close()
}

func publish_msg(msg amqp.Publishing) {
	err := ch.Publish(
		"",         // exchange
		queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

// Fiber Functions

func start_server() {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("publisher/index.html")
	})
	app.Post("/upload", upload)
	log.Fatal(app.Listen(":3000"))
}

func upload(c *fiber.Ctx) error {
	err := ioutil.WriteFile("images/sample.png", c.Body(), 0644)
	if err != nil {
		panic(err)
	}
	publish_msg(amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("images/sample.png"),
	})
	return c.SendString("ok")
}

// Helper Functions

func UNUSED(x ...interface{}) {}
