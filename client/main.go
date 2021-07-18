package main

import (
	"chat/builder"
	"chat/chat"
	"chat/gen"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"sync"
)

type Chat struct {
	sync.RWMutex
	started  bool
	finished bool
	me       *chat.User
	room     *chat.Room
	chat     *chat.Chat
}

func NewChat() *Chat {
	return &Chat{}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opts []grpc.DialOption
	// セキュア通信設定
	tls := false
	if tls {
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	cc, err := grpc.Dial("localhost:50051", opts...)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer cc.Close()

	c := NewChat()
	err = c.matching(ctx, gen.NewMatchingServiceClient(cc))
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Chat) matching(ctx context.Context, cli gen.MatchingServiceClient) error {
	stream, err := cli.JoinRoom(ctx, &gen.JoinRoomRequest{})
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	fmt.Println("Requested matching...")

	for {
		res, err := stream.Recv()
		if err != nil {
			return err
		}

		if res.GetStatus() == gen.JoinRoomResponse_MATCHED {
			c.room = builder.Room(res.GetRoom())
			c.me = builder.User(res.GetMe())
			fmt.Printf("Matched room_id=%v\n", res.GetRoom().GetId())
			return nil
		} else if res.GetStatus() == gen.JoinRoomResponse_WAITING {
			fmt.Println("Waiting matching...")
		} else {
			return fmt.Errorf("JoinRoomResponseStatus UNKNOWN error")
		}
	}
}
