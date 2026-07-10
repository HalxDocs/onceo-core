package main

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	onceo "github.com/HalxDocs/onceo-core"
	"github.com/HalxDocs/onceo-core/providers/paystack"
)

var store = onceo.NewMemoryStore()

func main() {
	r := gin.Default()

	r.POST("/v1/webhooks/paystack", handlePaystack)
	r.GET("/v1/events", listEvents)

	log.Println("listening on :8080")
	r.Run(":8080")
}

func handlePaystack(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read body"})
		return
	}
	defer c.Request.Body.Close()

	provider, err := paystack.New("sk_test_abc123")
	if err != nil {
		log.Printf("paystack.New: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provider init failed"})
		return
	}

	event, err := onceo.Process(c.Request.Context(), provider, store, c.Request.Header, body)
	if err != nil {
		switch {
		case errors.Is(err, onceo.ErrInvalidSignature):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		case errors.Is(err, onceo.ErrDuplicateEvent):
			c.JSON(http.StatusOK, gin.H{"status": "duplicate"})
		default:
			log.Printf("processing error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, event)
}

func listEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not yet implemented"})
}


