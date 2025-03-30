package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	litellmv1 "github.com/wagnerjt/go-mcp/webhook/litellm/v1"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("Received webhook: %s\n", string(body))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received successfully"))

	// proto.Marshal()

	u := &litellmv1.User{
		Id:   "12345",
		Name: "John Doe",
		Time: &timestamppb.Timestamp{
			Seconds: 1622547800,
		},
	}
	log.Printf("User ID: %s", u.GetId())
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)

	fmt.Println("Server listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
