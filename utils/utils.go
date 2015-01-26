package utils

import(
    "fmt"
    "io/ioutil"
    "sync"
    "regexp"
    "strings"
    "net/http"
    "log"
    "time"
    "strconv"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)


//Changes diacritics with no diacritic letter  
func FlattenWord(str string) string{
	rExps := map[string]string{
		"[\\xE0-\\xE6]" : "a",
		"[\\xE8-\\xEB]" : "e",
		"[\\xEC-\\xEF]" : "i",
		"[\\xF2-\\xF6]" : "o",
		"[\\xF9-\\xFC]" : "u",
		"[\\xF1]" : "n", 
	};

	str = strings.ToLower(str)

	for k, _ := range rExps{
		reg, _ := regexp.Compile(k)
		str = reg.ReplaceAllString(str, rExps[k])
	}

	return str
}


//HTTP GET common function
//if there is any error, it generates a new call to itself with some delay
//the responseconcat param is used to pass info in the response to control paginations if there is no other way
func Get(url string, responseconcat string, responses chan string, wg *sync.WaitGroup, delay *time.Duration) {
    res, err := http.Get(url)
    if err != nil {
        log.Println(err)
        time.Sleep(*delay * time.Millisecond)
        Get(url, responseconcat, responses, wg, delay)
    } else {
        defer res.Body.Close()
        body, err := ioutil.ReadAll(res.Body)
        if err != nil {
            log.Println(err)
            time.Sleep(*delay * time.Millisecond)
            Get(url, responseconcat, responses, wg, delay)
        } else {
            responses <- string(body) + responseconcat
        }
    }
}


//Updates data in mongodb replacing existing data in collection with the slice of objects provided
func UpdateMongo(url string, port int, database string, collection string, user string, password string, docitems []interface{}){

    start := time.Now()

    //insert into mongodb   
    session, err := mgo.Dial("mongodb://"+user+":"+password+"@"+url+":"+strconv.Itoa(port)+"/" + database)
    defer session.Close()

    if err == nil {
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(database).C(collection)
        c.RemoveAll(bson.M{}) //deletes all content before insert
        err = c.Insert(docitems...)
        if err != nil {
            fmt.Println(err)
        }
    }else{
        fmt.Println("err connecting to MongoDB")
        fmt.Println(err)
    }

    elapsed := time.Since(start)
    fmt.Printf("It took: %s insert into MongoDB\n", elapsed)
}

