package handler

import "chat/gen"

type ChatHandler struct {
}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

func (h *ChatHandler) Chat(stream gen.ChatService_ChatServer) error {
	return nil
}
