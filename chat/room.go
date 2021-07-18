package chat

type Room struct {
	ID    int32
	Host  *User
	Guest *User
}
