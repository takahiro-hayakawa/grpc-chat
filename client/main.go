package main

import (
	"bufio"
	"chat/builder"
	"chat/chat"
	"chat/gen"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"sync"
	"time"
)

type Chat struct {
	sync.RWMutex
	started  bool
	finished bool
	me       *chat.User
	room     *chat.Room
	message  *chat.Message
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

	err = c.chat(ctx, gen.NewChatServiceClient(cc))
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

func (c *Chat) chat(ctx context.Context, cli gen.ChatServiceClient) error {
	con, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := cli.Chat(con)
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	go func() {
		err := c.send(con, stream)
		if err != nil {
			cancel()
		}
	}()

	err = c.receive(con, stream)
	if err != nil {
		cancel()
		return err
	}

	return nil
}

func (c *Chat) send(ctx context.Context, stream gen.ChatService_ChatClient) error {
	for {
		c.RLock()

		if c.finished {
			// チャット終了
			c.RUnlock()
			return nil
		} else if !c.started {
			// チャット開始前
			err := stream.Send(&gen.ChatRequest{
				RoomId: c.room.ID,
				User:   builder.PBUser(c.me),
				Action: &gen.ChatRequest_Start{
					Start: &gen.ChatRequest_StartAction{},
				},
			})
			c.RUnlock()
			if err != nil {
				return err
			}

			for {
				// 相手が開始するまで待機
				c.RLock()
				if c.started {
					c.RUnlock()
					fmt.Println("READY GO!")
					break
				}
				c.RUnlock()
				fmt.Println("Waiting until opponent user ready")
				time.Sleep(1 * time.Second)
			}
		} else {
			// チャット中
			c.RUnlock()
			fmt.Print("Input Your Message:")
			stdin := bufio.NewScanner(os.Stdin)
			stdin.Scan()

			message := stdin.Text()

			go func() {
				err := stream.Send(&gen.ChatRequest{
					RoomId: c.room.ID,
					User:   builder.PBUser(c.me),
					Action: &gen.ChatRequest_Message{
						Message: &gen.ChatRequest_MessageAction{
							Message: &gen.Message{
								Message: message,
							}},
					},
				})
				if err != nil {
					fmt.Println(err)
				}
			}()

			// 一度メッセージを送ったら5秒間待機する
			ch := make(chan int)
			go func(ch chan int) {
				fmt.Println("")
				for i := 0; i < 5; i++ {
					fmt.Printf("freezing in %d second.\n", (5 - i))
					time.Sleep(1 * time.Second)
				}
				fmt.Println("")
				ch <- 0
			}(ch)
			<-ch
		}
		select {
		case <-ctx.Done():
			// キャンセルされたので終了する
			return nil
		default:
		}
	}
}

func (c *Chat) receive(ctx context.Context, stream gen.ChatService_ChatClient) error {
	for {
		// サーバーからのストリーミングを受け取る
		res, err := stream.Recv()
		if err != nil {
			return err
		}

		c.Lock()
		switch res.GetEvent().(type) {
		case *gen.ChatResponse_Waiting:
			// 開始待機中
		case *gen.ChatResponse_Ready:
			// 開始
			c.started = true
		case *gen.ChatResponse_Message:
			message := res.GetEvent().(*gen.ChatResponse_Message).Message.Message.GetMessage()
			// 相手のメッセージであれば表示する
			messageUserID := res.GetEvent().(*gen.ChatResponse_Message).Message.GetUser().GetId()
			if c.me.ID != messageUserID {
				fmt.Println("")
				fmt.Printf("Partner Message:%v\n", message)
				fmt.Print("Input Your Message:")
			}
		case *gen.ChatResponse_Finished:
			// チャットが終了した
			c.finished = true
			fmt.Println("Chat partner has left")
			// ループを終了する
			c.Unlock()
			return nil
		}
		c.Unlock()

		select {
		case <-ctx.Done():
			// キャンセルされたので終了する
			return nil
		default:
		}
	}
}
