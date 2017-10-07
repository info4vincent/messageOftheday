package main

import (
	"fmt"
	"os"
	"time"
	"strings"
	"crypto/tls"
	"gopkg.in/mgo.v2"
)

func TraverseDir(session *mgo.Session, dayName string) {
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

func main() {
	session := connectDB()

	defer session.Close()

	TraverseDir(session, "monday")
	
}