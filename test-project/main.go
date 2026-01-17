package main

import (
	"fmt"
	"time"

	userv1 "github.com/yourorg/yourproject/generated/go/v1/protos/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	// Create a new user using generated protobuf types
	user := &userv1.User{
		UserId:    "user-123",
		Email:     "test@example.com",
		Name:      "John Doe",
		Status:    userv1.UserStatus_USER_STATUS_ACTIVE,
		CreatedAt: timestamppb.New(time.Now()),
		UpdatedAt: timestamppb.New(time.Now()),
	}

	fmt.Printf("✅ Successfully created user:\n")
	fmt.Printf("   ID: %s\n", user.GetUserId())
	fmt.Printf("   Email: %s\n", user.GetEmail())
	fmt.Printf("   Name: %s\n", user.GetName())
	fmt.Printf("   Status: %s\n", user.GetStatus().String())
	fmt.Printf("   Created: %s\n", user.GetCreatedAt().AsTime().Format(time.RFC3339))

	// Test GetUserRequest
	req := &userv1.GetUserRequest{
		UserId: "user-456",
	}

	fmt.Printf("\n✅ GetUserRequest:\n")
	fmt.Printf("   UserID: %s\n", req.GetUserId())

	// Test ListUsersRequest
	listReq := &userv1.ListUsersRequest{
		PageSize:  10,
		PageToken: "next-page",
	}

	fmt.Printf("\n✅ ListUsersRequest:\n")
	fmt.Printf("   PageSize: %d\n", listReq.GetPageSize())
	fmt.Printf("   PageToken: %s\n", listReq.GetPageToken())

	fmt.Println("\n🎉 All proto types work correctly with googleapis dependencies!")
}
