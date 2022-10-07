package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var ch *amqp.Channel

// Main Function

func main() {
	init_connection()
	defer close_connection()
	start_consumer()
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
	fmt.Println("Connected to RabbitMQ")
}

func close_connection() {
	ch.Close()
	conn.Close()
}

func start_consumer() {
	msgs, err := ch.Consume(
		"test_queue", // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			fmt.Println("Received a message ...")
			filepath := string(d.Body)
			grayScaleImage(filepath)
		}
	}()
	fmt.Println("Successfully connected to our RabbitMQ instance")
	fmt.Println(" [*] - Waiting for messages. To exit press CTRL+C")
	<-forever
}

// Image Processing Function

func grayScaleImage(filepath string) {
	fmt.Println("Processing " + filepath + " ...")
	f, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	imgSrc, _, err := image.Decode(f)
	if err != nil {
		panic(err.Error())
	}

	// Create a new grayscale image
	bounds := imgSrc.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	grayScale := image.NewGray(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			imageColor := imgSrc.At(x, y)
			rr, gg, bb, _ := imageColor.RGBA()
			r := math.Pow(float64(rr), 2.2)
			g := math.Pow(float64(gg), 2.2)
			b := math.Pow(float64(bb), 2.2)
			m := math.Pow(0.2125*r+0.7154*g+0.0721*b, 1/2.2)
			Y := uint16(m + 0.5)
			grayColor := color.Gray{uint8(Y >> 8)}
			grayScale.Set(x, y, grayColor)
		}
	}

	// Encode the grayscale image to the new file
	newFileName := "images/result.png"
	newfile, err := os.Create(newFileName)
	if err != nil {
		log.Printf("failed creating %s: %s", newfile, err)
		panic(err.Error())
	}
	defer newfile.Close()
	png.Encode(newfile, grayScale)
	fmt.Println("File processed")
}
