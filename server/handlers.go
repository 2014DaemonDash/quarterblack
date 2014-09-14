package main

import (
	// "encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"code.google.com/p/go.crypto/bcrypt"
)

type User struct {
	Name    string
	Pass    []byte `json:"-"`
	Type    int    // 0 is Foodbank
	Address string
	Phone   string
}

type zipCodeReq struct {
	Zipcode string  `json:"zip_code"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	City    string  `json:"city"`
	State   string  `json:"state"`
}

// func (u *User) setPassword(pass string){
//   if u.Pass != nil {
//   password, err:=  bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
//     if err != nil {
//       //Bcrypt couldn't use default cost... What the....
//       log.Panicln("Something is wrong, Bcrypt couldn't hash like it wanted to")
//     }
//     u.Pass = passw
//   }
// }
//SetPassword sets the user password from a given input.
func (u *User) setPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Bcrypt couldn't use default cost. Password not updated.")
		return err
	}
	u.Pass = hashedPassword
	return err
}
func serveHomeFile(w http.ResponseWriter, req *http.Request) {
	log.Println("In ServeFile")
	http.ServeFile(w, req, "../static/html/home.html")
}

func serveSignUpFile(w http.ResponseWriter, req *http.Request) {
	log.Println("In Serve")
	http.ServeFile(w, req, "../static/html/signup.html")
}

func serveConsumerFile(w http.ResponseWriter, req *http.Request) {
	log.Println("In Serve")
	http.ServeFile(w, req, "../static/html/Consumer.html")
}
func serveFoodBankFile(w http.ResponseWriter, req *http.Request) {
	log.Println("In Serve")
	http.ServeFile(w, req, "../static/html/FoodBank.html")
}
func serveRestaurantFile(w http.ResponseWriter, req *http.Request) {
	log.Println("In Serve")
	http.ServeFile(w, req, "../static/html/Restaurant.html")
}
// func serveFolder(w http.ResponseWriter, req *http.Request){
// 	http.ServeFil
// }


//CheckPassword checks the given password against the hashed one.
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword(u.Pass, []byte(password))
}

func specSearchHandler(w http.ResponseWriter, req *http.Request) {

	params := mux.Vars(req)
	zipcode, err := strconv.Atoi(params["zipcode"])
	if err != nil {
		//Not a valid num, send malformed input error
	}
	log.Println(zipcode)
	// var search Search
	// ReadJSON(req, search)

}

func searchHandler(w http.ResponseWriter, req *http.Request) {}

func specUserHandler(w http.ResponseWriter, req *http.Request) {


	// zipURL := "http://zipcodedistanceapi.redline13.com/rest/FEB5PIgMRCAWte0YT4VORTZRWDfTWKeCpQIfzY1qACm4Rn3KrYyiRyaNbpH8aWLA/info.json/" + strconv.Itoa(zipcode) + "/degrees"
	// client := &http.Client{}
	// reqs, _ := http.NewRequest("GET", zipURL, nil)
	// res, err := client.Do(reqs)
	// if err != nil {
	// 	log.Println("Zip Code Request didn't go through properly.'")
	// }
	// defer res.Body.Close()
	// var data zipCodeReq
	// //Do this instead of io.ReadAll so we don't need contiguous mem
	// if json.NewDecoder(res.Body).Decode(&data) != nil {
	// 	log.Println("Malformed Data.")
	// }

}

func genUserHandler(w http.ResponseWriter, req *http.Request) {
}

func loginHandler(w http.ResponseWriter, req *http.Request) {
	name, password := req.FormValue("name"), req.FormValue("password")
	users, errs := SearchUserbyName(name, 0, 1)
	if errs != nil {
		//Something bad happened, return 500
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(users) < 1 {
		//No users found, 404?
	}
	errs = users[0].CheckPassword(password)
	if errs != nil {
		//Not authenticated return 401
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	tokenString, err := createUserToken(&users[0])
	if err != nil {
		http.Error(w, "Error while signing token", http.StatusInternalServerError)
		return
		//Something went wrong
	}

	ServeJSON(w, map[string]string{"jwt": tokenString})

}

func logoutHandler(w http.ResponseWriter, req *http.Request) {

}

func signupHandler(w http.ResponseWriter, req *http.Request) {
	var newUser User
	ReadJSON(req, newUser)
	//	name, password := req.FormValue("name"), req.FormValue("password"), req.FormValue("T")
	/*	newUser := User{
			//	ID:   bson.NewObjectId(),
			Name: name,
		}
	*/
	err := newUser.setPassword(string(newUser.Pass))
	if err != nil {
		//If bcrypt errored out, no business in making the user.
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	Insert("user", newUser)
	tokenString, err := createUserToken(&newUser)
	if err != nil {
		http.Error(w, "Error while signing token", http.StatusInternalServerError)
		return
		//Something went wrong
	}
	w.Header().Add("authorization", tokenString)
	if newUser.Type == 0 {
		http.Redirect(w, req, "/res", http.StatusFound)

	} else {
		http.Redirect(w, req, "/fb", http.StatusFound)
	}
}

//SearchUser th
func SearchUser(q interface{}, skip int, limit int) (searchResults []User, err error) {
	searchResults = []User{}
	query := func(c *mgo.Collection) error {
		fn := c.Find(q).Skip(skip).Limit(limit).All(&searchResults)
		if limit < 0 {
			fn = c.Find(q).Skip(skip).All(&searchResults)
		}
		return fn
	}
	search := func() error {
		return withCollection("user", query)
	}
	err = search()
	return
}

//SearchUserbyName is a generic form for searching for a User
//Set skip to zero is you want all the results, set limit to < 0  if you want all the results
//Naming the results allows us to not have to return them
func SearchUserbyName(name string, skip int, limit int) (searchResults []User, err error) {
	searchResults, err = SearchUser(bson.M{"name": name}, 0, -1)
	return
}
