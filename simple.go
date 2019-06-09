package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Database Models
// They come with ID, CreatedAt, UpdatedAt and DeletedAt for free

// Folder model for ORM
type Folder struct {
	gorm.Model
	Name string
}

// Record model for ORM
type Record struct {
	gorm.Model
	FolderID int
	Folder   Folder
	Key      string
	Data     []byte
}

func initDB() {
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Record{})
	db.AutoMigrate(&Folder{})

	// Apply extra rules

	// TODO: if we want cascase on delete folder
	//db.Model(&Profile{}).AddForeignKey("record_refer", "users(refer)", "CASCADE", "CASCADE")
}

// TODO: this could be a one liner...
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func getDBHandler() *gorm.DB {
	// TODO: always migrate until we have data we want to save
	os.Remove("test.db")

	// Check to see if we have a db already
	// TODO: right now we have to delete to migrate (i think)
	if !fileExists("test.db") {
		fmt.Println("TestDB doesnt exist, creating and migrating")
		initDB()
	}
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

func insertRecord(dbHandle *gorm.DB, record *Record) {
	dbHandle.Create(record)
}

func dbTesting() {
	db := getDBHandler()
	defer db.Close()
	// Create
	exampleData, _ := json.Marshal(map[string]interface{}{"Name": "Bob", "Food": "Pickle"})
	var exampleRecord = &Record{
		Folder: Folder{
			Name: "test",
		},
		Key:  "example",
		Data: exampleData,
	}
	insertRecord(db, exampleRecord)

	// TODO check searching
	// db.First(&product, "code = ?", "L1212") // find product with code l1212

	// Find the folder metadata associated with this
	var assocfolder Folder
	db.Model(exampleRecord).Related(&assocfolder)
	folderJ, _ := json.Marshal(assocfolder)
	fmt.Println("///Associated Folder ///")
	fmt.Println(string(folderJ))
	fmt.Println("///End Folder ///")
	// Update - update product's price to 2000
	// db.Model(&product).Update("Price", 2000)

	// Delete - delete product
	// db.Delete(ptrrecord)
	// db.Delete(ptrfolder)
}

// Samwise is our app
type Samwise struct {
	Router *mux.Router
	DB     *gorm.DB
}

// Initialize sets up DB conn for app and mux router
func (s *Samwise) Initialize() {
	//TODO: (user, password, dbname string)
	s.Router = mux.NewRouter()
	s.DB = getDBHandler()

	exampleData, _ := json.Marshal(map[string]interface{}{"Name": "Bob", "Food": "Pickle"})
	var exampleRecord = &Record{
		Folder: Folder{
			Name: "test",
		},
		Key:  "example",
		Data: exampleData,
	}
	insertRecord(s.DB, exampleRecord)

	s.DB.Create(&Folder{Name: "test2"})

	// TODO: i guess it automatically closes on finish
	//defer s.db.Close()
}

// Run function to host our application; starts the webserver
func (s *Samwise) Run(addr string) int {

	baseURL := "/api/v1"
	s.Router.HandleFunc(baseURL+"/keys/{folder}", s.handleKeysGet).Methods("GET")

	s.Router.HandleFunc(baseURL+"/{folder}/{key}", s.handleBasicGet).Methods("GET")
	s.Router.HandleFunc(baseURL+"/{folder}/{key}", s.handleBasicPost).Methods("POST")
	http.ListenAndServe(addr, s.Router)
	return 0
}

// GetResponse : used
type GetResponse struct {
	Query    interface{}
	Data     interface{}
	Success  bool
	Messages []string
}

// TODO: the top most level of keys dont get sorted

func respondWithJSON(w http.ResponseWriter, code int, payload GetResponse) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (s *Samwise) getMatchingFolderOrNil(name string, folder *Folder) error {
	return s.DB.Where("name = ?", name).Find(folder).Error
}

func (s *Samwise) handleKeysGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folder := vars["folder"]
	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested the folder: %s", folder))

	// Find folder to match...
	var matchedFolder Folder
	err := s.getMatchingFolderOrNil(folder, &matchedFolder)
	if err != nil {
		respondWithJSON(w, http.StatusNotFound, GetResponse{
			Query:    make(map[string]string),
			Data:     make(map[string]string),
			Success:  false,
			Messages: append(messages, "Folder not found"),
		})
		return
	}

	var records []Record
	// This needs to be {}
	// so json marshall will send back [] when empty
	keys := []string{}

	s.DB.Model(&matchedFolder).Related(&records)

	for _, record := range records {
		keys = append(keys, record.Key)
	}

	respondWithJSON(w, http.StatusOK, GetResponse{
		Query:    vars,
		Data:     keys,
		Success:  true,
		Messages: messages,
	})
}

func recordProcessMeta(record Record, meta string) interface{} {
	var output map[string]interface{}
	switch meta {
	case "only":
		recj, _ := json.Marshal(record)
		json.Unmarshal(recj, &output)
		delete(output, "Data")
		break
	case "on":
		// do nothing
		// output
		recj, _ := json.Marshal(record)
		json.Unmarshal(recj, &output)
		var dataj map[string]interface{}
		json.Unmarshal(record.Data, &dataj)
		output["Data"] = dataj
		break
	case "off":
		json.Unmarshal(record.Data, &output)
		break
	}
	return output
}

func (s *Samwise) handleBasicGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folder := vars["folder"]
	key := vars["key"]

	vars["meta"] = r.URL.Query()["meta"][0]

	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested the folder: %s with key %s", folder, key))

	// Find folder to match...
	var matchedFolder Folder
	fresult := s.DB.Where("name = ?", folder).Find(&matchedFolder)
	if fresult.Error != nil {
		respondWithJSON(w, http.StatusNotFound,
			GetResponse{
				Query:    vars,
				Data:     make(map[string]string),
				Success:  false,
				Messages: append(messages, "Folder not found"),
			})
		return
	}

	var record Record
	rresult := s.DB.Model(&matchedFolder).Where("key = ?", key).Find(&record)
	if rresult.Error != nil {
		respondWithJSON(w, http.StatusNotFound,
			GetResponse{
				Query:    vars,
				Data:     make(map[string]string),
				Success:  false,
				Messages: append(messages, "Record not found"),
			})
		return
	}

	output := recordProcessMeta(record, vars["meta"])

	respondWithJSON(w, http.StatusOK, GetResponse{
		Query:    vars,
		Data:     output,
		Success:  true,
		Messages: messages,
	})
}

func (s *Samwise) handleBasicPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folder := vars["folder"]
	key := vars["key"]

	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested the folder: %s with key %s", folder, key))

	var data interface{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		respondWithJSON(w, http.StatusBadRequest,
			GetResponse{
				Query:    vars,
				Data:     make(map[string]string),
				Success:  false,
				Messages: append(messages, "Invalid request payload"),
			})

		return
	}
	defer r.Body.Close()

	// TODO: actually create and add json to db
	// if err := p.createProduct(a.DB); err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, err.Error())
	// 	return
	// }
	respondWithJSON(w, http.StatusCreated, GetResponse{
		Query:    vars,
		Data:     data,
		Success:  true,
		Messages: append(messages, "Created Successfully"),
	})
}

func main() {
	s := Samwise{}
	s.Initialize()
	s.Run(":8080")
}
