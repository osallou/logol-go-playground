package transport

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	redis "github.com/go-redis/redis"
	logol "github.com/osallou/logol-go-playground/lib/types"
	"github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

var doStats bool

// TransportRabbit is the transport manager using RabbitMQ
type TransportRabbit struct {
	id     string
	conn   *amqp.Connection
	ch     *amqp.Channel
	queues map[int]string
	redis  *redis.Client
}

// GetQueueStatus returns number of pending messages and number of message consumer for the queue
func (t TransportRabbit) GetQueueStatus(queueID QueueType) (pending int, consumers int) {
	queueName := "analyse"
	switch queueID {
	case QUEUE_MESSAGE:
		queueName = "analyse"
	case QUEUE_RESULT:
		queueName = "result"
	case QUEUE_CASSIE:
		queueName = "cassie"
	}
	queue, _ := t.ch.QueueInspect("logol-" + queueName + "-" + t.id)
	pending += queue.Messages
	consumers += queue.Consumers
	return pending, consumers
}

// GetId returns the unique transport id
func (t TransportRabbit) GetID() string {
	return t.id
}

// GetProgress returns number of final matches, number of match under analysis and rejected matches
func (t *TransportRabbit) GetProgress(uid string) (count int, ban int, match int) {
	rcount, _ := t.redis.Get("logol:" + uid + ":count").Result()
	count, _ = strconv.Atoi(rcount)
	rban, _ := t.redis.Get("logol:" + uid + ":ban").Result()
	ban, _ = strconv.Atoi(rban)
	rmatch, _ := t.redis.Get("logol:" + uid + ":match").Result()
	match, _ = strconv.Atoi(rmatch)
	return count, ban, match
}

// AddBan reject a match
func (t *TransportRabbit) AddBan(uid string, nb int64) {
	t.redis.IncrBy("logol:"+uid+":ban", nb)
}

// AddCount increment number of solutions
func (t *TransportRabbit) AddCount(uid string, nb int64) {
	t.redis.IncrBy("logol:"+uid+":count", nb)
}

// AddMatch increment number of successful match
func (t *TransportRabbit) AddMatch(uid string, nb int64) {
	t.redis.IncrBy("logol:"+uid+":match", nb)
}

// SetBan sets the number of ban for result
func (t *TransportRabbit) SetBan(uid string, nb int64) {
	t.redis.Set("logol:"+uid+":ban", nb, 0)
}

// SetCount sets the number of result
func (t *TransportRabbit) SetCount(uid string, nb int64) {
	t.redis.Set("logol:"+uid+":count", nb, 0)
}

// SetMatch sets the number of match for result
func (t *TransportRabbit) SetMatch(uid string, nb int64) {
	t.redis.Set("logol:"+uid+":match", nb, 0)
}

// Clear redis info for the run id
func (t *TransportRabbit) Clear(uid string) {
	t.redis.Del("logol:" + uid + ":count")
	t.redis.Del("logol:" + uid + ":match")
	t.redis.Del("logol:" + uid + ":ban")
	t.redis.Del("logol:" + uid + ":grammar")
	t.redis.Del("logol:" + uid + ":stat:duration")
	t.redis.Del("logol:" + uid + ":stat:flow")
	t.redis.Del("logol:" + uid + ":toban")
}

// SetGrammar saves grammar in redis
func (t *TransportRabbit) SetGrammar(grammar []byte, grammarID string) (err bool) {
	logger.Debugf("Set grammar %s", grammarID)
	t.redis.Set("logol:"+grammarID+":grammar", grammar, 0)
	return true
}

// GetGrammar fetch grammar from redis
func (t *TransportRabbit) GetGrammar(grammarID string) (logol.Grammar, bool) {
	logger.Debugf("Get grammar %s", grammarID)
	grammar, err := t.redis.Get(grammarID).Result()
	if grammar == "" {
		logger.Errorf("Failed to get grammar %s", grammarID)
		return logol.Grammar{}, true
	}
	err, g := logol.LoadGrammar([]byte(grammar))
	if err != nil {
		logger.Errorf("error: %v", err)
		return logol.Grammar{}, true
	}
	return g, false
}

// Close cleanup queues
func (t *TransportRabbit) Close() {
	logger.Debugf("Closing transport %s", t.id)
	t.ch.ExchangeDelete("logol-event-exchange-"+t.id, false, false)
	t.ch.QueueDelete("logol-analyse-"+t.id, false, false, false)
	t.ch.QueueDelete("logol-result-"+t.id, false, false, false)
	t.ch.QueueDelete("logol-cassie-"+t.id, false, false, false)
	t.ch.QueueDelete("logol-log-"+t.id, false, false, false)
	t.ch.Close()
	t.conn.Close()
}

// Init setup queues
func (t *TransportRabbit) Init(uid string) {
	doStatsEnv := os.Getenv("LOGOL_STATS")
	if doStatsEnv != "" {
		doStats = true
		logger.Infof("Activating usage statistics, this can impact performance")
	}
	t.id = uid
	queueName := t.id
	rabbitConURL := "amqp://guest:guest@localhost:5672"
	osRabbitConURL := os.Getenv("LOGOL_RABBITMQ_ADDR")
	if osRabbitConURL != "" {
		rabbitConURL = osRabbitConURL
	}
	//connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
	//    h.User, h.Password, h.Hostname, h.Port)
	conn, err := amqp.Dial(rabbitConURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	t.conn = conn
	//defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	t.ch = ch
	//defer ch.Close()

	t.queues = make(map[int]string)

	qLog, lerr := ch.QueueDeclare(
		"logol-log-"+queueName, // name
		false,                  // durable
		false,                  // delete when usused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)

	failOnError(lerr, "Failed to declare a queue")
	t.queues[QUEUE_LOG] = qLog.Name

	err = ch.ExchangeDeclare(
		"logol-event-exchange-"+queueName, // name
		"fanout",                          // kind
		false,                             // durable
		false,                             // delete when usused
		false,                             // exclusive
		false,                             // no-wait
		nil,                               // arguments
	)

	failOnError(err, "Failed to declare an exchange")

	eventQueue, err := ch.QueueDeclare(
		"",
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)

	failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(
		eventQueue.Name,                   // name,
		"",                                // key
		"logol-event-exchange-"+queueName, // exchange
		false,                             // no-wait
		nil,                               // arguments
	)

	failOnError(err, "Failed to bind queue")

	t.queues[QUEUE_EVENT] = eventQueue.Name
	t.queues[EXCHANGE_EVENT] = "logol-event-exchange-" + queueName

	qAnalyse, err := ch.QueueDeclare(
		"logol-analyse-"+queueName, // name
		false,                      // durable
		false,                      // delete when usused
		false,                      // exclusive
		false,                      // no-wait
		nil,                        // arguments
	)

	failOnError(err, "Failed to declare a queue")

	t.queues[QUEUE_MESSAGE] = qAnalyse.Name

	qCassie, cerr := ch.QueueDeclare(
		"logol-cassie-"+queueName, // name
		false,                     // durable
		false,                     // delete when usused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)

	failOnError(cerr, "Failed to declare a queue")

	t.queues[QUEUE_CASSIE] = qCassie.Name

	qResult, rqerr := ch.QueueDeclare(
		"logol-result-"+queueName, // name
		false,                     // durable
		false,                     // delete when usused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)

	failOnError(rqerr, "Failed to declare a queue")

	t.queues[QUEUE_RESULT] = qResult.Name

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")
}

// ListenLog starts a loop waiting for log message. On message, callback function is called
func (t *TransportRabbit) ListenLog(fn CallbackLog) {
	queueListenName, ok := t.queues[QUEUE_LOG]
	if !ok {
		panic(fmt.Sprintf("%s", "Failed to find log queue name"))
	}
	logger.Infof("Listen on queue %s", queueListenName)
	eventQueueName, eok := t.queues[QUEUE_EVENT]
	if !eok {
		panic(fmt.Sprintf("%s", "Failed to find event queue name"))
	}

	msgs, err := t.ch.Consume(
		queueListenName, // queue
		"",              // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to register a consumer")

	events, err := t.ch.Consume(
		eventQueueName, // queue
		"",             // consumer
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)

	failOnError(err, "Failed to register a consumer")

	wg := sync.WaitGroup{}
	wg.Add(2)
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			result := string(d.Body[:])
			fn(result)
			d.Ack(false)
		}
		wg.Done()
	}()

	go func(ch chan bool) {
		for d := range events {
			logger.Debugf("New message on %s, %s", queueListenName, string(d.Body[:]))
			msgEvent := MsgEvent{}
			json.Unmarshal([]byte(d.Body), &msgEvent)
			switch msgEvent.Step {
			case STEP_END:
				logger.Infof("Received exit request %s", queueListenName)
				d.Ack(false)
				wg.Done()
				ch <- true
			default:
				d.Ack(false)
			}
		}
	}(forever)

	logger.Infof(" [*] Waiting for logs.")
	<-forever
	t.ch.Close()
	t.conn.Close()
	wg.Wait()
}

// Listen starts a loop waiting for message on selected queue. On message, callback function is called
func (t *TransportRabbit) Listen(queueListen QueueType, fn CallbackMessage) {

	queueListenName, ok := t.queues[int(queueListen)]
	if !ok {
		panic(fmt.Sprintf("%s", "Failed to find message queue name"))
		//Errorf("Failed to find message queue %d", int(queueListen))
	}
	logger.Debugf("Listen on queue %s", queueListenName)
	eventQueueName, eok := t.queues[QUEUE_EVENT]
	if !eok {
		panic(fmt.Sprintf("%s", "Failed to find event queue name"))
	}

	msgs, err := t.ch.Consume(
		queueListenName, // queue
		"",              // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to register a consumer")

	events, err := t.ch.Consume(
		eventQueueName, // queue
		"",             // consumer
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)

	failOnError(err, "Failed to register a consumer")

	wg := sync.WaitGroup{}
	wg.Add(2)
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			logger.Debugf("New message on %s, %s", queueListenName, string(d.Body[:]))
			result, _ := t.getMessage(string(d.Body[:]))
			fn(result)
			d.Ack(false)
		}
		wg.Done()
	}()

	go func(ch chan bool) {
		for d := range events {
			logger.Debugf("New message on %s, %s", queueListenName, string(d.Body[:]))
			msgEvent := MsgEvent{}
			json.Unmarshal([]byte(d.Body), &msgEvent)
			switch msgEvent.Step {
			case STEP_END:
				logger.Infof("Received exit request %s", queueListenName)
				d.Ack(false)
				wg.Done()
				ch <- true
			default:
				d.Ack(false)
			}
		}
	}(forever)

	logger.Infof(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	t.ch.Close()
	t.conn.Close()
	wg.Wait()
}

func (t TransportRabbit) getMessage(uid string) (result logol.Result, err error) {
	// fetch from redis the message based on provided uid
	// Once fetched, delete it from db
	val, err := t.redis.Get("logol:msg:" + uid).Result()
	if err == redis.Nil {
		return logol.Result{}, err
	}
	result = logol.Result{}
	json.Unmarshal([]byte(val), &result)
	t.redis.Del(uid)
	return result, err
}

// SendLog sends a log message to log queue
func (t TransportRabbit) SendLog(msg string) bool {
	queueName, ok := t.queues[QUEUE_LOG]
	if !ok {
		logger.Errorf("Could not find queue %d", QUEUE_LOG)
		return false
	}
	publishMsg := amqp.Publishing{}
	publishMsg.Body = []byte(msg)
	t.ch.Publish(
		"",        // exchange
		queueName, // key
		false,     // mandatory
		false,     // immediate
		publishMsg,
	)
	return true
}

// PrepareMessage store data in redis and returns its identifier
func (t TransportRabbit) PrepareMessage(data logol.Result) string {
	u1 := uuid.Must(uuid.NewV4())

	jsonMsg, _ := json.Marshal(data)
	err := t.redis.Set("logol:msg:"+u1.String(), jsonMsg, 0).Err()
	if err != nil {
		logger.Errorf("Failed to store message")
	}

	return u1.String()
}

// PublishMessage post a msg to a rabbitmq queue
func (t TransportRabbit) PublishMessage(queue string, msg string) {
	publishMsg := amqp.Publishing{}
	publishMsg.Body = []byte(msg)
	t.ch.Publish(
		"",    // exchange
		queue, // key
		false, // mandatory
		false, // immediate
		publishMsg,
	)
}

// PublishExchange post a msg to a rabbitmq exchange
func (t TransportRabbit) PublishExchange(exchange string, msg string) {
	publishMsg := amqp.Publishing{}
	publishMsg.Body = []byte(msg)
	t.ch.Publish(
		exchange, // exchange
		"",       // key
		false,    // mandatory
		false,    // immediate
		publishMsg,
	)
}

// SendEvent sends an *MsgEvent* to exchange
func (t TransportRabbit) SendEvent(event MsgEvent) bool {
	queueExchange, _ := t.queues[int(EXCHANGE_EVENT)]
	//publish_msg := amqp.Publishing{}
	jsonMsg, _ := json.Marshal(event)
	//publish_msg.Body = []byte(json_msg)
	t.PublishExchange(queueExchange, string(jsonMsg))
	return true
}

// SendMessage sends a new message to defined queue
func (t TransportRabbit) SendMessage(queue QueueType, data logol.Result) bool {
	queueName, ok := t.queues[int(queue)]
	if !ok {
		logger.Errorf("Could not find queue %d", int(queue))
		return false
	}
	publishMsg := t.PrepareMessage(data)
	if queue == QUEUE_EVENT {
		queueExchange, _ := t.queues[int(EXCHANGE_EVENT)]
		logger.Debugf("Send message to event exchange")
		t.PublishExchange(queueExchange, publishMsg)
	} else {
		logger.Debugf("Send message to %s", queueName)
		t.PublishMessage(queueName, publishMsg)
	}
	return true
}

// IncrFlowStat increments number of calls between 2 variables
func (t TransportRabbit) IncrFlowStat(uid string, from string, to string) {
	if !doStats || from == "." || to == "over.over" {
		return
	}
	err := t.redis.HIncrBy("logol:"+uid+":stat:flow", from+"."+to, 1).Err()
	if err != nil {
		logger.Errorf("Failed to update stats")
	}
}

// IncrDurationStat increments duration for defined variable
func (t TransportRabbit) IncrDurationStat(uid string, variable string, duration int64) {
	if !doStats {
		return
	}
	err := t.redis.HIncrBy("logol:"+uid+":stat:duration", variable, duration).Err()
	if err != nil {
		logger.Errorf("Failed to update stats")
	}
}

// Stats information
type Stats struct {
	Duration map[string]string
	Flow     map[string]string
}

// GetStats get workflow informations (links and duration)
func (t TransportRabbit) GetStats(uid string) Stats {
	statDuration, errD := t.redis.HGetAll("logol:" + uid + ":stat:duration").Result()
	statFlow, errF := t.redis.HGetAll("logol:" + uid + ":stat:flow").Result()
	stats := Stats{}
	if errD == nil {
		stats.Duration = statDuration
	}
	if errF == nil {
		stats.Flow = statFlow
	}
	return stats
}

// AddToBan appends the variable uid to the list of matches to filter out from result
func (t TransportRabbit) AddToBan(uid string, varUID string) {
	t.redis.LPush("logol:"+uid+":toban", varUID)
}

// GetToBan returns the list of variable uids to be filterd out from result
func (t TransportRabbit) GetToBan(uid string) (toBan []string) {
	nbElts, err := t.redis.LLen("logol:" + uid + ":toban").Result()
	if err != nil {
		return toBan
	}
	for v := int64(0); v < nbElts; v++ {
		matchToBan, err := t.redis.LPop("logol:" + uid + ":toban").Result()
		if err == nil {
			toBan = append(toBan, matchToBan)
		}
	}
	return toBan
}

func newRedisClient() (client *redis.Client) {
	redisAddr := "localhost:6379"
	osRedisAddr := os.Getenv("LOGOL_REDIS_ADDR")
	if osRedisAddr != "" {
		redisAddr = osRedisAddr
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := redisClient.Ping().Result()
	if err != nil {
		logger.Errorf("Failed to contact Redis database")
	}
	return redisClient
}

// NewTransportRabbit create a new transport instance
func NewTransportRabbit() *TransportRabbit {
	transport := TransportRabbit{}

	transport.redis = newRedisClient()

	return &transport
}
