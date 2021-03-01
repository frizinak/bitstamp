package ws

type Subscribe struct {
	Event string `json:"event"`
	Data  struct {
		Channel Channel `json:"channel"`
	} `json:"data"`
}

func newSubscribe(event string, channel Channel) (s Subscribe) {
	s.Event = event
	s.Data.Channel = channel
	return s
}

func NewSubscribe(channel Channel) Subscribe   { return newSubscribe("bts:subscribe", channel) }
func NewUnsubscribe(channel Channel) Subscribe { return newSubscribe("bts:unsubscribe", channel) }
