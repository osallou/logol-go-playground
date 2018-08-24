package logol

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "strconv"
    "sync"
    "github.com/streadway/amqp"
    "github.com/satori/go.uuid"
    logol "org.irisa.genouest/logol/lib/types"
    redis "github.com/go-redis/redis"
)

type TransportRabbit struct {
    id string
    conn *amqp.Connection
    ch   *amqp.Channel
    queues map[int]string
    redis *redis.Client
}

func (t TransportRabbit) GetId() string {
    return t.id
}

func(t *TransportRabbit) GetProgress(uid string) (count int, ban int, match int) {
    rcount, _ := t.redis.Get("logol:" + uid + ":count").Result()
    count, _ = strconv.Atoi(rcount)
    rban, _ := t.redis.Get("logol:" + uid + ":ban").Result()
    ban, _ = strconv.Atoi(rban)
    rmatch, _ := t.redis.Get("logol:" + uid + ":match").Result()
    match, _ = strconv.Atoi(rmatch)
    return count, ban, match
}

func(t *TransportRabbit) AddBan(uid string, nb int64) {
    t.redis.IncrBy("logol:" + uid + ":ban", nb)
}
func(t *TransportRabbit) AddCount(uid string, nb int64) {
    t.redis.IncrBy("logol:" + uid + ":count", nb)
}
func(t *TransportRabbit) AddMatch(uid string, nb int64) {
    t.redis.IncrBy("logol:" + uid + ":match", nb)
}
func(t *TransportRabbit) SetBan(uid string, nb int64) {
    t.redis.Set("logol:" + uid + ":ban", nb, 0)
}
func(t *TransportRabbit) SetCount(uid string, nb int64) {
    t.redis.Set("logol:" + uid + ":count", nb, 0)
}
func(t *TransportRabbit) SetMatch(uid string, nb int64) {
    t.redis.Set("logol:" + uid + ":match", nb, 0)
}
func(t *TransportRabbit) Clear(uid string) {
    t.redis.Del("logol:" + uid + ":count")
    t.redis.Del("logol:" + uid + ":match")
    t.redis.Del("logol:" + uid + ":ban")
    t.redis.Del("logol:" + uid + ":grammar")
}

func(t *TransportRabbit) SetGrammar(grammarFile string, grammarId string) (err bool){
    logger.Infof("Set grammar %s, %s", grammarFile, grammarId)
    grammar, _ := ioutil.ReadFile(grammarFile)
    t.redis.Set("logol:" + grammarId + ":grammar", grammar, 0)
    return true
}
func(t *TransportRabbit) GetGrammar(grammarId string) (logol.Grammar, bool) {
    logger.Infof("Get grammar %s", grammarId)
    grammar, err := t.redis.Get(grammarId).Result()
    if grammar == "" {
        logger.Errorf("Failed to get grammar %s", grammarId)
        return logol.Grammar{}, true
    }
    err, g := logol.LoadGrammar([]byte(grammar))
    if err != nil {
            logger.Errorf("error: %v", err)
            return logol.Grammar{}, true
    }
    return g, false
}

func (t *TransportRabbit) Close() {
    logger.Infof("Closing transport %s", t.id)
    t.ch.ExchangeDelete("logol-event-exchange-" + t.id, false, false)
    t.ch.QueueDelete("logol-analyse-" + t.id, false, false, false)
    t.ch.QueueDelete("logol-result-" + t.id, false, false, false)
    t.ch.QueueDelete("logol-cassie-" + t.id, false, false, false)
    t.ch.Close()
    t.conn.Close()
}

func(transport *TransportRabbit) Init(uid string){
    transport.id = uid
    queueName := transport.id
    rabbitConUrl := "amqp://guest:guest@localhost:5672"
    osRabbitConUrl := os.Getenv("LOGOL_RABBITMQ_ADDR")
    if osRabbitConUrl != "" {
        rabbitConUrl = osRabbitConUrl
    }
    //connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
    //    h.User, h.Password, h.Hostname, h.Port)
    conn, err := amqp.Dial(rabbitConUrl)
    failOnError(err, "Failed to connect to RabbitMQ")
    transport.conn = conn
    //defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    transport.ch = ch
    //defer ch.Close()

    transport.queues = make(map[int]string)

    err = ch.ExchangeDeclare(
      "logol-event-exchange-" + queueName, // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare an exchange")

    eventQueue, err := ch.QueueDeclare(
        "",
        false,   // durable
        false,   // delete when usused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        eventQueue.Name, // name,
        "", // key
        "logol-event-exchange-" + queueName,  // exchange
        false, // no-wait
        nil, // arguments
    )

    failOnError(err, "Failed to bind queue")

    transport.queues[QUEUE_EVENT] = eventQueue.Name
    transport.queues[EXCHANGE_EVENT] = "logol-event-exchange-" + queueName


    qAnalyse, err := ch.QueueDeclare(
      "logol-analyse-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    transport.queues[QUEUE_MESSAGE] = qAnalyse.Name

    qCassie, cerr := ch.QueueDeclare(
      "logol-cassie-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(cerr, "Failed to declare a queue")

    transport.queues[QUEUE_CASSIE] = qCassie.Name


    qResult, rqerr := ch.QueueDeclare(
      "logol-result-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(rqerr, "Failed to declare a queue")

    transport.queues[QUEUE_RESULT] = qResult.Name

    err = ch.Qos(
      1,     // prefetch count
      0,     // prefetch size
      false, // global
    )
    failOnError(err, "Failed to set QoS")
}

func(s *TransportRabbit) Listen(queueListen QueueType, fn CallbackMessage) {

    queueListenName, ok := s.queues[int(queueListen)]
    if ! ok {
        panic(fmt.Sprintf("%s", "Failed to find message queue name"))
        //Errorf("Failed to find message queue %d", int(queueListen))
    }
    logger.Errorf("Listen on queue %s", queueListenName)
    eventQueueName, eok := s.queues[QUEUE_EVENT]
    if ! eok {
        panic(fmt.Sprintf("%s", "Failed to find event queue name"))
    }

    msgs, err := s.ch.Consume(
      queueListenName, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    events, err := s.ch.Consume(
      eventQueueName, // queue
      "",     // consumer
      false,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )

    failOnError(err, "Failed to register a consumer")

    wg := sync.WaitGroup{}
    wg.Add(2)
    forever := make(chan bool)
    go func() {
        for d := range msgs {
            logger.Debugf("New message on %s, %s", queueListenName, string(d.Body[:]))
            result, _ := s.getMessage(string(d.Body[:]))
            fn(result)
            d.Ack(false)
        }
        wg.Done()
    }()

    go func(ch chan bool) {
      for d := range events {
          logger.Infof("New message on %s, %s", queueListenName, string(d.Body[:]))
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
    s.ch.Close()
    s.conn.Close()
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


func (t TransportRabbit) PrepareMessage(data logol.Result) string {
    u1 := uuid.Must(uuid.NewV4())

    json_msg, _ := json.Marshal(data)
    err := t.redis.Set("logol:msg:" + u1.String(), json_msg, 0).Err()
    if err != nil{
        logger.Errorf("Failed to store message")
    }

    return u1.String()
}
func (s TransportRabbit) PublishMessage(queue string, msg string) {
    publish_msg := amqp.Publishing{}
    publish_msg.Body = []byte(msg)
    s.ch.Publish(
        "", // exchange
        queue, // key
        false, // mandatory
        false, // immediate
        publish_msg,
    )
}
func (s TransportRabbit) PublishExchange(queue string, msg string) {
    publish_msg := amqp.Publishing{}
    publish_msg.Body = []byte(msg)
    s.ch.Publish(
        queue, // exchange
        "", // key
        false, // mandatory
        false, // immediate
        publish_msg,
    )
}

func (s TransportRabbit) SendEvent(event MsgEvent) bool {
    queueExchange, _ := s.queues[int(EXCHANGE_EVENT)]
    //publish_msg := amqp.Publishing{}
    json_msg, _ := json.Marshal(event)
    //publish_msg.Body = []byte(json_msg)
    s.PublishExchange(queueExchange, string(json_msg))
    return true
}
func (s TransportRabbit) SendMessage(queue QueueType, data logol.Result) bool {
    queueName, ok := s.queues[int(queue)]
    if ! ok {
        logger.Errorf("Could not find queue %d", int(queue))
        return false
    }
    publish_msg := s.PrepareMessage(data)
    if queue == QUEUE_EVENT {
        queueExchange, _ := s.queues[int(EXCHANGE_EVENT)]
        logger.Infof("Send message to event exchange")
        s.PublishExchange(queueExchange, publish_msg)
    }else {
        logger.Infof("Send message to %s", queueName)
        s.PublishMessage(queueName, publish_msg)
    }
    return true
}


func newRedisClient() (client *redis.Client){
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

	pong, err := redisClient.Ping().Result()
	fmt.Println(pong, err)
	return redisClient
}

func NewTransportRabbit() (*TransportRabbit){
    transport := TransportRabbit{}
    //transport.id = queueName

    transport.redis = newRedisClient()

    /*
    rabbitConUrl := "amqp://guest:guest@localhost:5672"
    osRabbitConUrl := os.Getenv("LOGOL_RABBITMQ_ADDR")
    if osRabbitConUrl != "" {
        rabbitConUrl = osRabbitConUrl
    }
    //connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
    //    h.User, h.Password, h.Hostname, h.Port)
    conn, err := amqp.Dial(rabbitConUrl)
    failOnError(err, "Failed to connect to RabbitMQ")
    transport.conn = conn
    //defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    transport.ch = ch
    //defer ch.Close()

    transport.queues = make(map[int]string)

    err = ch.ExchangeDeclare(
      "logol-event-exchange-" + queueName, // name
      "fanout",  // kind
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare an exchange")

    eventQueue, err := ch.QueueDeclare(
        "",
        false,   // durable
        false,   // delete when usused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        eventQueue.Name, // name,
        "", // key
        "logol-event-exchange-" + queueName,  // exchange
        false, // no-wait
        nil, // arguments
    )

    failOnError(err, "Failed to bind queue")

    transport.queues[QUEUE_EVENT] = eventQueue.Name
    transport.queues[EXCHANGE_EVENT] = "logol-event-exchange-" + queueName


    qAnalyse, err := ch.QueueDeclare(
      "logol-analyse-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(err, "Failed to declare a queue")

    transport.queues[QUEUE_MESSAGE] = qAnalyse.Name

    qCassie, cerr := ch.QueueDeclare(
      "logol-cassie-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(cerr, "Failed to declare a queue")

    transport.queues[QUEUE_CASSIE] = qCassie.Name


    qResult, rqerr := ch.QueueDeclare(
      "logol-result-" + queueName, // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    failOnError(rqerr, "Failed to declare a queue")

    transport.queues[QUEUE_RESULT] = qResult.Name

    err = ch.Qos(
      1,     // prefetch count
      0,     // prefetch size
      false, // global
    )
    failOnError(err, "Failed to set QoS")
    */

    return &transport
}
