package main

import (
        "encoding/json"
        "io/ioutil"
        "log"
        "net/http"
        "strconv"
        "time"

        jwt "github.com/dgrijalva/jwt-go"
        "github.com/gorilla/mux"
        "gopkg.in/mgo.v2"
)

/*
Home
About
Partners
Contact Us
Donate / Support
Share
#Search <- Important
*/

var (
        privateKey []byte
        publicKey  []byte
        s          = Server{}
        //RestMethods that are accesible
        RestMethods = []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "PATCH"}
        routes      = Routes{
                {
                        PrefixRoute: "/search",
                        PostfixRoute: []string{
                                "/{zipcode:[0-9]+}",
                        },
                        PrefixHandler: searchHandler,
                        PostfixHandler: []func(http.ResponseWriter, *http.Request){
                                specSearchHandler,
                        },
                },
                {
                        PrefixRoute: "/user",
                        PostfixRoute: []string{
                                "/{name:[\\w]+}",
                        },
                        PrefixHandler: genUserHandler,
                        PostfixHandler: []func(http.ResponseWriter, *http.Request){
                                specUserHandler,
                        },
                },
                {
                        PrefixRoute:   "/login",
                        PrefixHandler: loginHandler,
                },
                {
                        PrefixRoute:   "/register",
                        PrefixHandler: signupHandler,
                },
                {
                        PrefixRoute:   "/logout",
                        PrefixHandler: logoutHandler,
                },
        }
)

//Server for database managment and such
type Server struct {
        Session *mgo.Session
        dbName  string
        DBURI   string
        Routes  Routes
}

//Route ...
type Route struct {
        PrefixRoute    string
        PostfixRoute   []string
        PrefixHandler  func(w http.ResponseWriter, req *http.Request)
        PostfixHandler []func(w http.ResponseWriter, req *http.Request)
}

//Routes ...
type Routes []Route

//Search ...
type Search struct {
}

func main() {
        initServer()
        //Don't close session till end of main block, which doesn't occur
        //until the server itself is killed
        defer s.Session.Close()
        http.Handle("/", s.initHandlers())
        // http.Handle("/static/", http.FileServer(http.Dir("../static")))
        log.Println("Quarterblack DaemonDash 2014. Now Listening...")
        log.Fatalln(http.ListenAndServe(":8080", nil))
}

func initServer() {
        s.initDB()
        privateKey, _ = ioutil.ReadFile("./daemondash.rsa")
        publicKey, _ = ioutil.ReadFile("./daemondash.rsa.pub")
        s.Routes = routes
}

func (s *Server) initDB() {
        s.DBURI = "localhost"
        s.dbName = "daemondash"
        s.getSession()
        // Ensure that any query that changes data is processed without error
        //Set to nil for faster throughput but no error checking
        s.Session.SetSafe(&mgo.Safe{})
        s.Session.SetMode(mgo.Monotonic, true)
        /*      cNames, errors := EnsureIndex(CollectionNames, Indices...)
                for k, err := range errors {
                        if err != nil {
                                log.Printf("Can't assert index for %v;%v\n", cNames[k], err)
                        }
                }
        */
}

//EnsureIndex makes sure our rules about a colelction are enforced.
func EnsureIndex(collectionNames []string, indices ...mgo.Index) (s []string, e []error) {
        for k, i := range indices {
                fn := func(c *mgo.Collection) error {
                        return c.EnsureIndex(i)
                }
                err := withCollection(collectionNames[k], fn)
                if err != nil {
                        s = append(s, collectionNames[k])
                        e = append(e, err)
                }
        }
        return
}

func (s *Server) initHandlers() *mux.Router {
        r := mux.NewRouter()
        for _, value := range s.Routes {
                router := r.PathPrefix(value.PrefixRoute).Subrouter()
                router.HandleFunc("/", value.PrefixHandler).Methods(RestMethods...).Name(value.PrefixRoute)
                for k, i := range value.PostfixHandler {
                        router.HandleFunc(value.PostfixRoute[k], i).Methods(RestMethods...).Name(value.PostfixRoute[k])
                }
        }
        initFileHandlers(r)
        return r
}

func initFileHandlers(r *mux.Router) {
        // route := r.PathPRefix("/static").Subrouter()
        // route.HandleFunc("/", serveStaticFolder)

        router := r.PathPrefix("/home").Subrouter()
        router.HandleFunc("/", serveHomeFile)
        rs := r.PathPrefix("/signup").Subrouter()
        rs.HandleFunc("/", serveSignUpFile)
        d := r.PathPrefix("/consumer").Subrouter()
        d.HandleFunc("/", serveConsumerFile)

}

//ServeJSON Serves JSON
func ServeJSON(w http.ResponseWriter, v interface{}) {
        if d, err := json.Marshal(v); err != nil {
                log.Panicln("Marshalling Err.")
                http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
                w.Header().Set("Content-Length", strconv.Itoa(len(d)))
                w.Header().Set("Content-Type", "application/json; charset=utf-8")
                w.Write(d)
        }

}

// ReadJSON decodes JSON data into a provided struct which must be passed in as a pointer.
func ReadJSON(req *http.Request, v interface{}) error {
        defer req.Body.Close()
        decoder := json.NewDecoder(req.Body)
        err := decoder.Decode(v)
        return err
}

//Use this method to debug things
func logRequest(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
        return func(w http.ResponseWriter, r *http.Request) {
                var s = time.Now()
                handler(w, r)
                log.Printf("%s %s %6.3fms", r.Method, r.RequestURI, (time.Since(s).Seconds() * 1000))
        }
}

func (s *Server) getSession() *mgo.Session {
        if s.Session == nil {
                var err error
                di := &mgo.DialInfo{
                        Addrs:    []string{s.DBURI},
                        Direct:   true,
                        FailFast: true, //You may want to turn this off if you're expecting latency
                }
                s.Session, err = mgo.DialWithInfo(di)
                if err != nil {
                        log.Fatalf("Can't find Mongodb.\n Ensure that it is running and you have the correct address., %v\n", err)
                }
        }
        //If you also want to reuse the socket, use clone instead
        return s.Session.Copy()
}

//WithCollection takes the name of the collection, along with a function
//that expects the connection object to that collection,
//and can execute access functions on it.
func withCollection(collection string, fn func(*mgo.Collection) error) error {
        session := s.getSession()
        defer session.Close()
        c := session.DB(s.dbName).C(collection)
        return fn(c)
}

//Insert x amount of data into a collection
func Insert(collectionName string, values ...interface{}) error {
        fn := func(c *mgo.Collection) error {
                err := c.Insert(values...)
                if err != nil {
                        log.Printf("Can't insert/update document, %v\n", err)
                }
                return err
        }
        return withCollection(collectionName, fn)
}

//Serve405 serves a 405 Method Not Allowed error while attatching the required allow header.
func Serve405(w http.ResponseWriter, allow string) {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.Header().Set("Allow", allow)
        w.WriteHeader(http.StatusMethodNotAllowed)
        w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
}

func createUserToken(u *User) (string, error) {
        //Generate a JWT, pass it along.
        token := jwt.New(jwt.GetSigningMethod("HS256"))
        // Set some claims
        token.Claims["usr"] = u.Name
        token.Claims["type"] = u.Type
        token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
        // Sign and get the complete encoded token as a string
        tokenString, e := token.SignedString(privateKey)
        if e != nil {
                log.Printf("Token Signing error: %v\n", e)
                return "", e
        }
        return tokenString, e
}

func authRequest(w http.ResponseWriter, r *http.Request) {
        token, err := jwt.Parse(w.Header().Get("authorization"),
                func(*jwt.Token) (interface{}, error) {
                        return publicKey, nil
                },
        )
        if err != nil {
                //Do things that matter
        }

        if token.Valid {
                //Pass in the next ahndler.
                //YAY!
        } else {
                //Someone is being funny
        }
}

//func ( ) keyFun() ([]byte, error) {
//      return publicKey, nil
//}
