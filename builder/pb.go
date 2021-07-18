package builder

import (
	"chat/chat"
	"chat/gen"
)

func PBRoom(r *chat.Room) *gen.Room {
	return &gen.Room{
		Id:    r.ID,
		Host:  PBUser(r.Host),
		Guest: PBUser(r.Guest),
	}
}

func PBUser(r *chat.User) *gen.User {
	return &gen.User{
		Id: r.ID,
	}
}
