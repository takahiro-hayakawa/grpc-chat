package handler

import (
	"chat/builder"
	"chat/chat"
	"chat/gen"
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

type MatchingHandler struct {
	sync.RWMutex
	Rooms     map[int32]*chat.Room
	maxUserID int32
}

func NewMatchingHandler() *MatchingHandler {
	return &MatchingHandler{
		Rooms: make(map[int32]*chat.Room),
	}
}

func (h *MatchingHandler) JoinRoom(req *gen.JoinRoomRequest, stream gen.MatchingService_JoinRoomServer) error {
	ctx, cancel := context.WithTimeout(stream.Context(), 2*time.Minute)
	defer cancel()

	h.Lock()
	h.maxUserID++
	me := &chat.User{
		ID: h.maxUserID,
	}

	for _, room := range h.Rooms {
		if room.Guest == nil {
			room.Guest = me
			stream.Send(&gen.JoinRoomResponse{
				Status: gen.JoinRoomResponse_MATCHED,
				Room:   builder.PBRoom(room),
				Me:     builder.PBUser(room.Guest),
			})
			h.Unlock()
			fmt.Printf("matched room_id=%v\n", room.ID)
			return nil
		}
	}

	room := &chat.Room{
		ID:   int32(len(h.Rooms)) + 1,
		Host: me,
	}
	h.Rooms[room.ID] = room
	stream.Send(&gen.JoinRoomResponse{
		Status: gen.JoinRoomResponse_WAITING,
		Room:   builder.PBRoom(room),
	})
	h.Unlock()

	ch := make(chan int)
	go func(ch chan<- int) {
		for {
			h.RLock()
			guest := room.Guest
			h.RUnlock()

			if guest != nil {
				stream.Send(&gen.JoinRoomResponse{
					Status: gen.JoinRoomResponse_MATCHED,
					Room:   builder.PBRoom(room),
					Me:     builder.PBUser(room.Host),
				})
				ch <- 0
				break
			}
			time.Sleep(1 * time.Second)

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(ch)

	select {
	case <-ch:
	case <-ctx.Done():
		return status.Errorf(codes.DeadlineExceeded, "マッチングできませんでした")
	}
	return nil
}
