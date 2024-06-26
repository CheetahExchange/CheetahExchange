package pushing

import (
	"context"
	"encoding/json"
	"github.com/CheetahExchange/CheetahExchange/service"
	"github.com/gorilla/websocket"
	"github.com/siddontang/go-log/log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512

	writeChannelSize    = 256
	l2ChangeChannelSize = 512
)

var id int64

// Each connection corresponds to a client, which is responsible for the data I/O of the connection.
type Client struct {
	id         int64
	conn       *websocket.Conn
	writeCh    chan interface{}
	l2ChangeCh chan *Level2Change
	sub        *subscription
	channels   map[string]struct{}
	mu         sync.Mutex
}

func NewClient(conn *websocket.Conn, sub *subscription) *Client {
	return &Client{
		id:         atomic.AddInt64(&id, 1),
		conn:       conn,
		writeCh:    make(chan interface{}, writeChannelSize),
		l2ChangeCh: make(chan *Level2Change, l2ChangeChannelSize),
		sub:        sub,
		channels:   map[string]struct{}{},
	}
}

func (c *Client) startServe() {
	go c.runReader()
	go c.runWriter()
}

func (c *Client) runReader() {
	defer func() {
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Error(err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.close()
			break
		}

		var req Request
		err = json.Unmarshal(message, &req)
		if err != nil {
			log.Errorf("bad message : %v %v", string(message), err)
			c.close()
			break
		}

		c.onMessage(&req)
	}
}

func (c *Client) runWriter() {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		cancel()
		ticker.Stop()
		_ = c.conn.Close()
	}()

	go c.runL2ChangeWriter(ctx)

	for {
		select {
		case message := <-c.writeCh:
			// Forward l2change messages for incremental push
			switch message.(type) {
			case *Level2Change:
				c.l2ChangeCh <- message.(*Level2Change)
				continue
			}

			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.close()
				return
			}

			buf, err := json.Marshal(message)
			if err != nil {
				continue
			}
			err = c.conn.WriteMessage(websocket.TextMessage, buf)
			if err != nil {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.close()
				return
			}

		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.close()
				return
			}

			err = c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.close()
				return
			}
		}
	}
}

func (c *Client) runL2ChangeWriter(ctx context.Context) {
	type state struct {
		resendSnapshot bool
		changes        []*Level2Change
		lastSeq        int64
	}
	states := map[string]*state{}

	stateOf := func(productId string) *state {
		s, found := states[productId]
		if found {
			return s
		}
		s = &state{
			resendSnapshot: true,
			changes:        nil,
			lastSeq:        0,
		}
		states[productId] = s
		return s
	}

	for {
		select {
		case <-ctx.Done():
			return
		case l2Change := <-c.l2ChangeCh:
			state := stateOf(l2Change.ProductId)

			if state.resendSnapshot || l2Change.Seq == 0 {
				snapshot := getLastLevel2Snapshot(l2Change.ProductId)
				if snapshot == nil {
					log.Warnf("no snapshot for %v", l2Change.ProductId)
					continue
				}

				// Latest snapshot version is too old, discarded, waiting for newer snapshot version
				if state.lastSeq > snapshot.Seq {
					log.Warnf("last snapshot too old: %v changeSeq=%v snapshotSeq=%v",
						l2Change.ProductId, state.lastSeq, snapshot.Seq)
					continue
				}

				state.lastSeq = snapshot.Seq
				state.resendSnapshot = false

				c.writeCh <- &Level2SnapshotMessage{
					Type:      Level2TypeSnapshot,
					ProductId: l2Change.ProductId,
					Bids:      snapshot.Bids,
					Asks:      snapshot.Asks,
				}
				continue
			}

			// Discard changes where seq is less than snapshot seq
			if l2Change.Seq <= state.lastSeq {
				log.Infof("discard l2changeSeq=%v snapshotSeq=%v", l2Change.Seq, state.lastSeq)
				continue
			}

			// seq is not contiguous, message loss occurred, resend snapshot
			if l2Change.Seq != state.lastSeq+1 {
				log.Infof("l2change lost newSeq=%v lastSeq=%v", l2Change.Seq, state.lastSeq)
				state.resendSnapshot = true
				state.changes = nil
				state.lastSeq = l2Change.Seq
				if len(c.l2ChangeCh) == 0 {
					c.l2ChangeCh <- &Level2Change{ProductId: l2Change.ProductId}
				}
				continue
			}

			state.lastSeq = l2Change.Seq
			state.changes = append(state.changes, l2Change)

			// If chan still has messages, continue to read the buffer.
			if len(c.l2ChangeCh) > 0 && len(state.changes) < 10 {
				continue
			}

			updateMsg := &Level2UpdateMessage{
				Type:      Level2TypeUpdate,
				ProductId: l2Change.ProductId,
			}
			for _, change := range state.changes {
				updateMsg.Changes = append(updateMsg.Changes, [3]interface{}{change.Side, change.Price, change.Size})
			}
			c.writeCh <- updateMsg
			state.changes = nil
		}
	}
}

func (c *Client) onMessage(req *Request) {
	switch req.Type {
	case "subscribe":
		c.onSub(req.CurrencyIds, req.ProductIds, req.Channels, req.Token)
	case "unsubscribe":
		c.onUnSub(req.CurrencyIds, req.ProductIds, req.Channels, req.Token)
	default:
	}
}

func (c *Client) onSub(currencyIds []string, productIds []string, channels []string, token string) {
	user, err := service.CheckToken(token)
	if err != nil {
		log.Error(err)
		return
	}

	var userId int64
	if user != nil {
		userId = user.Id
	}

	for range currencyIds {
		for _, channel := range channels {
			switch Channel(channel) {
			case ChannelFunds:
				c.subscribe(ChannelFunds.FormatWithUserId(userId))
			}
		}
	}

	for _, productId := range productIds {
		for _, channel := range channels {
			switch Channel(channel) {
			case ChannelLevel2:
				if c.subscribe(ChannelLevel2.FormatWithProductId(productId)) {
					//if len(c.l2ChangeCh) == 0 {
					//	c.l2ChangeCh <- &Level2Change{ProductId: productId}
					//}
					c.l2ChangeCh <- &Level2Change{ProductId: productId}
				}

			case ChannelMatch:
				c.subscribe(ChannelMatch.FormatWithProductId(productId))

			case ChannelTrade:
				c.subscribe(ChannelTrade.Format(productId, user.Id))

			case ChannelTicker:
				if c.subscribe(ChannelTicker.FormatWithProductId(productId)) {
					ticker := getLastTicker(productId)
					if ticker != nil {
						c.writeCh <- ticker
					}
				}

			case ChannelOrder:
				c.subscribe(ChannelOrder.Format(productId, userId))

			case ChannelCandles1m:
				c.subscribe(ChannelCandles1m.FormatWithProductId(productId))

			case ChannelCandles3m:
				c.subscribe(ChannelCandles3m.FormatWithProductId(productId))

			case ChannelCandles5m:
				c.subscribe(ChannelCandles5m.FormatWithProductId(productId))

			case ChannelCandles15m:
				c.subscribe(ChannelCandles15m.FormatWithProductId(productId))

			case ChannelCandles30m:
				c.subscribe(ChannelCandles30m.FormatWithProductId(productId))

			case ChannelCandles60m:
				c.subscribe(ChannelCandles60m.FormatWithProductId(productId))

			case ChannelCandles120m:
				c.subscribe(ChannelCandles120m.FormatWithProductId(productId))

			case ChannelCandles240m:
				c.subscribe(ChannelCandles240m.FormatWithProductId(productId))

			case ChannelCandles360m:
				c.subscribe(ChannelCandles360m.FormatWithProductId(productId))

			case ChannelCandles720m:
				c.subscribe(ChannelCandles720m.FormatWithProductId(productId))

			case ChannelCandles1440m:
				c.subscribe(ChannelCandles1440m.FormatWithProductId(productId))

			default:
				continue
			}
		}
	}
}

func (c *Client) onUnSub(currencyIds []string, productIds []string, channels []string, token string) {
	user, err := service.CheckToken(token)
	if err != nil {
		log.Error(err)
		return
	}

	var userId int64
	if user != nil {
		userId = user.Id
	}

	for range currencyIds {
		for _, channel := range channels {
			switch Channel(channel) {
			case ChannelFunds:
				c.unsubscribe(ChannelFunds.FormatWithUserId(userId))
			}
		}
	}

	for _, productId := range productIds {
		for _, channel := range channels {
			switch Channel(channel) {
			case ChannelLevel2:
				c.unsubscribe(ChannelLevel2.FormatWithProductId(productId))

			case ChannelMatch:
				c.unsubscribe(ChannelMatch.FormatWithProductId(productId))

			case ChannelTrade:
				c.unsubscribe(ChannelTrade.Format(productId, user.Id))

			case ChannelTicker:
				c.unsubscribe(ChannelTicker.FormatWithProductId(productId))

			case ChannelOrder:
				c.unsubscribe(ChannelOrder.Format(productId, userId))

			case ChannelCandles1m:
				c.unsubscribe(ChannelCandles1m.FormatWithProductId(productId))

			case ChannelCandles3m:
				c.unsubscribe(ChannelCandles3m.FormatWithProductId(productId))

			case ChannelCandles5m:
				c.unsubscribe(ChannelCandles5m.FormatWithProductId(productId))

			case ChannelCandles15m:
				c.unsubscribe(ChannelCandles15m.FormatWithProductId(productId))

			case ChannelCandles30m:
				c.unsubscribe(ChannelCandles30m.FormatWithProductId(productId))

			case ChannelCandles60m:
				c.unsubscribe(ChannelCandles60m.FormatWithProductId(productId))

			case ChannelCandles120m:
				c.unsubscribe(ChannelCandles120m.FormatWithProductId(productId))

			case ChannelCandles240m:
				c.unsubscribe(ChannelCandles240m.FormatWithProductId(productId))

			case ChannelCandles360m:
				c.unsubscribe(ChannelCandles360m.FormatWithProductId(productId))

			case ChannelCandles720m:
				c.unsubscribe(ChannelCandles720m.FormatWithProductId(productId))

			case ChannelCandles1440m:
				c.unsubscribe(ChannelCandles1440m.FormatWithProductId(productId))

			default:
				continue
			}
		}
	}
}

func (c *Client) subscribe(channel string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, found := c.channels[channel]
	if found {
		return false
	}

	if c.sub.subscribe(channel, c) {
		c.channels[channel] = struct{}{}
		return true
	}
	return false
}

func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sub.unsubscribe(channel, c) {
		delete(c.channels, channel)
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for channel := range c.channels {
		c.sub.unsubscribe(channel, c)
	}
}
