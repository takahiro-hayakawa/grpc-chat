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

func PBUser(u *chat.User) *gen.User {
	if u == nil {
		return nil
	}

	return &gen.User{
		Id: u.ID,
	}
}

func Room(r *gen.Room) *chat.Room {
	return &chat.Room{
		ID:    r.Id,
		Host:  User(r.Host),
		Guest: User(r.Guest),
	}
}

func User(u *gen.User) *chat.User {
	return &chat.User{
		ID: u.Id,
	}
}
