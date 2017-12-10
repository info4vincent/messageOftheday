package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	zmq "github.com/pebbe/zmq4"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type WeekPlan struct {
	_id     string
	Userid  string
	Eventid string
	Action  string
	Data    string
}

var MyTestEvent = WeekPlan{_id: "10", Userid: "testEvent", Eventid: "testEvent1", Data: "test the data"}

const DBName = "mybluemarvin"
const CollectionName = "weekplan"

type handler func(w http.ResponseWriter, r *http.Request, db *mgo.Database)

func writeToDb(session *mgo.Session, eventinWeek WeekPlan) {
	c := session.DB(DBName).C(CollectionName)

	_, err := c.UpsertId(eventinWeek._id, &eventinWeek)
	if err != nil {
		log.Fatal(err)
	}
}

func GetMessageOfUserForEvent(session *mgo.Session, user string, eventId string) WeekPlan {
	c := session.DB(DBName).C(CollectionName)

	var results []WeekPlan
	err := c.Find(nil).All(&results)
	err = c.Find(bson.M{"userid": user, "eventid": eventId}).All(&results)

	log.Println("aantal results:", len(results))
	// actionUri := fmt.Sprintf("Say:Hoi, ik heb geen message gevonden voor user:%v en eventId", user, eventId)
	if err == nil {
		if len(results) > 0 {
			log.Println("Messagefound found returning details now.")
			return results[0]
		} else {
			log.Println("Could not find the user or event..")
			return WeekPlan{user, user, eventId, "Say", "user en of event niet gevonden."}
		}
	} else if err != nil {
		log.Println("Could not find user or event")
		return WeekPlan{user, user, eventId, "Say", "user en of event niet gevonden."}
	}

	return results[0]
}

func connectDB() *mgo.Session {
	uri := os.Getenv("MONGODB_URL")
	if uri == "" {
		fmt.Println("No connection string provided - set MONGODB_URL = mongodb://{user}:{password}@mongodb.documents.azure.com:{port}")
		os.Exit(1)
	}
	uri = strings.TrimSuffix(uri, "?ssl=true")

	tlsConfig := &tls.Config{}
	tlsConfig.InsecureSkipVerify = true

	dialInfo, err := mgo.ParseURL(uri)

	if err != nil {
		fmt.Println("Failed to parse URI: ", err)
		os.Exit(1)
	}

	maxWait := time.Duration(5 * time.Second)
	dialInfo.Timeout = maxWait

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		fmt.Println("Failed to connect: ", err)
		os.Exit(1)
	}

	dbnames, err := session.DB("").CollectionNames()
	if err != nil {
		fmt.Println("Couldn't query for collections names: ", err)
		os.Exit(1)
	}

	fmt.Println(dbnames)

	return session
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := session.Clone()
	defer s.Close()

	h(w, r, s.DB(DBName))
}

func GetMessage(w http.ResponseWriter, r *http.Request, db *mgo.Database) {
	params := mux.Vars(r)
	userid := params["userid"]
	eventid := params["eventid"]

	weekplan := GetMessageOfUserForEvent(db.Session, userid, eventid)
	fmt.Fprintf(w, "%s", weekplan.Data)
}

var session *mgo.Session

func main() {
	session = connectDB()

	defer session.Close()

	// writeToDb(session, MyTestEvent)

	dayPlan := GetMessageOfUserForEvent(session, "ruben", "welcome")

	fmt.Println(dayPlan.Data)

	fmt.Println("starting messageOftheday....")
	fmt.Println("Connecting to eventsource server...")
	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		log.Fatal(err)
	}

	defer requester.Close()
	err = requester.Connect("tcp://localhost:5555")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Client bind *:5555 succesful")

	myint, error := requester.Send("zmqmsgoftheday available..", 0)
	if error != nil {
		log.Println("Not send any message", error)
	} else {
		log.Println(myint)
	}

	event, error := requester.Recv(0)

	if error != nil {
		log.Println("Not received any..")
	} else {
		log.Println(event)
	}

	//  Socket to talk to server
	fmt.Println("Collecting updates from weather server...")
	subscriber, _ := zmq.NewSocket(zmq.SUB)
	defer subscriber.Close()
	subscriber.Connect("tcp://localhost:5556")

	//  Subscribe to Everything which a command to messageOftheday things..
	filter := "Messageoftheday:"
	subscriber.SetSubscribe(filter)

	// Wait for messages
	fmt.Println("MessageoftheDay is waiting for weather server...")
	for {
		event, error := subscriber.Recv(zmq.DONTWAIT)

		if error != nil {
			continue
		}
		
		
		eventData := strings.TrimPrefix(event, "Messageoftheday:")
		eventDataFields := strings.Fields(eventData)
		userID := eventDataFields[0]
	
		dayOfTheWeek := time.Now().Weekday().String()
		fmt.Println("Day of the week", dayOfTheWeek)

		eventID := strings.Replace(eventDataFields[1], "{dayoftheweek}", dayOfTheWeek, -1)
		
		msgoftheDayData := GetMessageOfUserForEvent(session, userID, eventID )
		// send reply back to client
		reply := fmt.Sprintf("%v:%v", msgoftheDayData.Action, msgoftheDayData.Data)
		log.Println("Sending:", reply)
		_, err = requester.Send(reply, 0)
		if err != nil {
			log.Println("Error while sending..", err)
		}
		_, err = requester.Recv(0)
		if err != nil {
			log.Println("Error while receiving..", err)
		}
		log.Println("publishing:", reply)
		subscriber.Send(reply, 0)
		fmt.Println("MessageoftheDay is waiting for weather server...")
	}

	// router := mux.NewRouter()
	// router.Handle("/message/{userid}/{eventid}", handler(GetMessage))

	// srv := &http.Server{
	// 	Handler: router,
	// 	Addr:    "127.0.0.1:8080",
	// 	// Good practice: enforce timeouts for servers you create!
	// 	WriteTimeout: 15 * time.Second,
	// 	ReadTimeout:  15 * time.Second,
	// }

	// log.Fatal(srv.ListenAndServe())
}
