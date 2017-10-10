package main

import (
	"net/http"
	"log"
	"fmt"
	"os"
	"time"
	"strings"
	"crypto/tls"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/gorilla/mux"
)

type WeekPlan struct {
	_id	        string
	Userid      string
	Eventid     string
	Action      string
	Data 		string
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
	err =c.Find(bson.M{"userid": user, "eventid": eventId}).All(&results)

	if (err == nil) {
		return results[0]
	} else if (err != nil ) {
		log.Fatal(err)
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

	writeToDb(session, MyTestEvent)

	dayPlan := GetMessageOfUserForEvent(session, "isis", "welcome")
	
	fmt.Println(dayPlan.Data)
	
	router := mux.NewRouter()
	router.Handle("/message/{userid}/{eventid}", handler(GetMessage))

	srv := &http.Server{
        Handler:      router,
        Addr:         "127.0.0.1:8080",
        // Good practice: enforce timeouts for servers you create!
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
}