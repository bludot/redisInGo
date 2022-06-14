package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

func unsubscribe(ctx context.Context, client *redis.PubSub) {
	client.Unsubscribe(ctx, "new_users")
}

func handleSubscribe(ctx context.Context, c *fiber.Ctx, client *redis.Client) error {

	// call redis if this c.Params("key") exists, if it does return

	// Subscribe to the Topic given
	topic := subscribe(ctx, client)
	defer unsubscribe(ctx, topic)
	// mark account as dirty
	_, err := http.Get("http://publisher/" + c.Params("key"))

	if err != nil {
		log.Fatalln(err)
	}
	channel := topic.Channel()
	for msg := range channel {
		u := &User{}
		// Unmarshal the data into the user
		err := u.UnmarshalBinary([]byte(msg.Payload))
		if err != nil {
			panic(err)
		}

		fmt.Println(u)
		if u.Key == c.Params("key") {
			c.SendString(u.Username + " is here!")
			return nil
		}
	}
	return nil
}

func subscribe(ctx context.Context, client *redis.Client) *redis.PubSub {
	topic := client.Subscribe(ctx, "new_users")
	// Get the Channel to use
	// Itterate any messages sent on the channel
	return topic
}

func main() {
	// Create a new Redis Client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",  // We connect to host redis, thats what the hostname of the redis service is set to in the docker-compose
		Password: "superSecret", // The password IF set in the redis Config file
		DB:       0,
	})
	// Ping the Redis server and check if any errors occured
	err := redisClient.Ping(context.Background()).Err()
	if err != nil {
		// Sleep for 3 seconds and wait for Redis to initialize
		time.Sleep(3 * time.Second)
		err := redisClient.Ping(context.Background()).Err()
		if err != nil {
			panic(err)
		}
	}
	ctx := context.Background()

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
	app.Get("/user/:key", func(c *fiber.Ctx) error {
		return handleSubscribe(ctx, c, redisClient)
	})

	app.Listen(":3000")

}

// User is a struct representing newly registered users
type User struct {
	Username string
	Email    string
	Key      string
}

// MarshalBinary encodes the struct into a binary blob
// Here I cheat and use regular json :)
func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

// UnmarshalBinary decodes the struct into a User
func (u *User) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, u); err != nil {
		return err
	}
	return nil
}

func (u *User) String() string {
	return "User: " + u.Username + " registered with Email: " + u.Email
}
