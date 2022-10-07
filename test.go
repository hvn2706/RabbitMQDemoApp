package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/streadway/amqp"
)

type Data struct {
	name string `json:"name"`
	done bool   `json:"done"`
}

var myQueue = make(map[string]Data)

func grayScaleImage(c *fiber.Ctx) error {
	name := c.Params("name")
	addToQueue(name)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")

	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer ch.Close()

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

	go func() {
		for message := range msgs {
			name := string(message.Body)
			log.Printf("Processing: %s", name)

			//Decode the image
			imgSrc, _, err := image.Decode(bytes.NewReader(c.Body()))
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
			newFileName := name
			newfile, err := os.Create(newFileName)
			if err != nil {
				log.Printf("failed creating %s: %s", newfile, err)
				panic(err.Error())
			}
			defer newfile.Close()
			png.Encode(newfile, grayScale)
			log.Printf("Done: %s", name)
		}
	}()
	return c.SendString("Image uploaded successfully!")
}

func addToQueue(name string) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")

	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
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

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(name),
		},
	)

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	fmt.Println("Added to queue: " + name)

	myQueue[name] = Data{name: name, done: false}
}

func getQueue(c *fiber.Ctx) error {
	return c.JSON(myQueue)
}

func getGrayImage(c *fiber.Ctx) error {
	name := c.Params("name")
	if checkFileExists(name) {
		return c.Download(name)
	}
	return c.SendString("Image not found")
}

func checkFileExists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}

func main() {

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("index.html")
	})

	app.Post("/upload/:name", grayScaleImage)
	app.Get("/queue", getQueue)
	app.Get("/image/:name", getGrayImage)

	log.Fatal(app.Listen(":3000"))
}
