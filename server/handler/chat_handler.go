package handler

import (
	"chat/builder"
	"chat/chat"
	"chat/gen"
	"fmt"
	"sync"
)

type ChatHandler struct {
	sync.RWMutex
	chats  map[int32][]*chat.Message
	client map[int32][]gen.ChatService_ChatServer
}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		client: make(map[int32][]gen.ChatService_ChatServer),
	}
}

func (h *ChatHandler) Chat(stream gen.ChatService_ChatServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		roomID := req.GetRoomId()
		user := builder.User(req.GetUser())

		switch req.GetAction().(type) {
		case *gen.ChatRequest_Start:
			err = h.start(stream, roomID)
			if err != nil {
				return err
			}
		case *gen.ChatRequest_Message:
			action := req.GetMessage()
			message := action.Message.Message
			err = h.sendMessage(roomID, user, message)
			if err != nil {
				return err
			}
		case *gen.ChatRequest_End:
			err = h.end(roomID)
			if err != nil {
				return err
			}
		}
	}
}

func (h *ChatHandler) start(stream gen.ChatService_ChatServer, roomID int32) error {
	h.Lock()
	defer h.Unlock()

	h.client[roomID] = append(h.client[roomID], stream)

	if len(h.client[roomID]) == 2 {
		for _, s := range h.client[roomID] {
			err := s.Send(&gen.ChatResponse{
				Event: &gen.ChatResponse_Ready{
					Ready: &gen.ChatResponse_ReadyEvent{},
				},
			})
			if err != nil {
				return err
			}
		}
		fmt.Printf("chat has started room_id=%v\n", roomID)
	} else {
		err := stream.Send(&gen.ChatResponse{
			Event: &gen.ChatResponse_Waiting{
				Waiting: &gen.ChatResponse_WaitingEvent{},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *ChatHandler) sendMessage(roomID int32, u *chat.User, message string) error {
	h.Lock()
	defer h.Unlock()

	for _, s := range h.client[roomID] {
		err := s.Send(&gen.ChatResponse{
			Event: &gen.ChatResponse_Message{
				Message: &gen.ChatResponse_MessageEvent{
					User: builder.PBUser(u),
					Message: &gen.Message{
						Message: message,
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *ChatHandler) end(roomID int32) error {
	h.Lock()
	defer h.Unlock()

	for _, s := range h.client[roomID] {
		err := s.Send(&gen.ChatResponse{
			Event: &gen.ChatResponse_Finished{
				Finished: &gen.ChatResponse_FinishedEvent{},
			},
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("chat has ended room_id=%v\n", roomID)

	return nil
}
