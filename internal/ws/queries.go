package ws

// When loading a room’s chat history, you can implement pagination to avoid sending too many messages at once.
// Load the most recent 50 messages, and if the user scrolls up, fetch more.
func GetMessages(h *Hub, roomID string, limit int) []*Message {
	rows, err := h.DB.Query("SELECT * FROM messages WHERE room_id = $1 ORDER BY timestamp DESC LIMIT $2", roomID, limit)
	if err != nil {
		// Handle err
	}

	var messages []*Message
	for rows.Next() {
		var msg Message
		err = rows.Scan(&msg.ID, &msg.RoomID, &msg.UserID, &msg.Content, &msg.Timestamp)

		messages = append(messages, &msg)
	}

	return messages
}
