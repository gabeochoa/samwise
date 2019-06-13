package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	Name       string
	Expiration time.Duration
}

// Record model for ORM
type Record struct {
	gorm.Model
	FolderID    int
	Folder      Folder
	Key         string
	Data        []byte
	LastTouched time.Time
}

// Now model represents what the current data looks like
type Now struct {
	gorm.Model
	RecordID int
	Record   Record
	FolderID int
	Folder   Folder
	Key      string
	Hash     []byte
}

// EventType which represents what kind of event happened
type EventType int

const (
	// NoChange ...
	NoChange EventType = iota
	// Insert ...
	Insert
	// Update ...
	Update
)

func eventTypeToString(event EventType) string {
	switch event {
	case NoChange:
		return "No Change"
		break
	case Insert:
		return "Insert"
		break
	case Update:
		return "Update"
		break
	}
	return "Unsupported Event"
}

// Event ...
type Event struct {
	gorm.Model
	RecordID  int
	Record    Record
	EventType EventType
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
	db.AutoMigrate(&Now{})
	db.AutoMigrate(&Event{})

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
	s.Router.HandleFunc(baseURL+"/folders", s.handleFoldersGet).Methods("GET")
	s.Router.HandleFunc(baseURL+"/folders/{folder}", s.handleFoldersPost).Methods("POST")
	s.Router.HandleFunc(baseURL+"/keys/{folder}", s.handleKeysGet).Methods("GET")
	s.Router.HandleFunc(baseURL+"/{folder}/{key}", s.handleBasicGet).Methods("GET")
	s.Router.HandleFunc(baseURL+"/{folder}/{key}", s.handleBasicPost).Methods("POST")
	http.ListenAndServe(addr, RequestLogger(s.Router))
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

func (s *Samwise) handleFoldersPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folder := vars["folder"]

	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested to create the folder: %s", folder))

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

	var existingFolder Folder
	var countFolder int
	_ = (s.DB.
		Where("name = ?", folder).
		Find(&existingFolder).
		Count(&countFolder))

	if countFolder != 0 {
		messages = append(messages, fmt.Sprintf("Folder with name %s already exists", folder))
		respondWithJSON(w, http.StatusBadRequest,
			GetResponse{
				Query:    vars,
				Data:     make(map[string]string),
				Success:  false,
				Messages: messages,
			})
		return
	}

	result := s.DB.Create(&Folder{
		Name: folder,
	})
	if result.Error != nil {
		messages = append(messages, fmt.Sprint(result.Error))
		respondWithJSON(w, http.StatusCreated, GetResponse{
			Query:    vars,
			Data:     data,
			Success:  false,
			Messages: append(messages, "Failed to create"),
		})
		return
	}

	respondWithJSON(w, http.StatusCreated, GetResponse{
		Query:    vars,
		Data:     data,
		Success:  true,
		Messages: append(messages, "Created Successfully"),
	})
}

func (s *Samwise) handleFoldersGet(w http.ResponseWriter, r *http.Request) {
	var folders []Folder
	s.DB.Find(&folders)
	respondWithJSON(w, http.StatusOK, GetResponse{
		Query: map[string]string{
			"folders": "all",
		},
		Data:     folders,
		Success:  true,
		Messages: make([]string, 0),
	})
}

func (s *Samwise) handleKeysGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folder := vars["folder"]
	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested the folder: %s", folder))

	// Find folder to match...
	var matchedFolder Folder
	if err := s.getMatchingFolderOrNil(folder, &matchedFolder); err != nil {
		respondWithJSON(w, http.StatusNotFound, GetResponse{
			Query:    make(map[string]string),
			Data:     make(map[string]string),
			Success:  false,
			Messages: append(messages, "Folder not found"),
		})
		return
	}

	var records []Record
	s.DB.Model(&matchedFolder).Related(&records)
	// This needs to be {}
	// so json marshall will send back [] when empty
	keys := []string{}

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

	queryParams := r.URL.Query()

	if val, ok := queryParams["meta"]; ok {
		vars["meta"] = val[0]
	} else {
		defaultMeta := "off"
		vars["meta"] = defaultMeta
	}

	messages := []string{}
	messages = append(messages, fmt.Sprintf("You've requested the folder: %s with key %s", folder, key))

	// Find folder to match...
	var matchedFolder Folder
	if fresult := s.DB.Where("name = ?", folder).Find(&matchedFolder); fresult.Error != nil {
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
	if s.DB.Model(&matchedFolder).Where("key = ?", key).Find(&record).Error != nil {
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
	r.Body.Close()

	var matchedFolder Folder
	if err := s.getMatchingFolderOrNil(folder, &matchedFolder); err != nil {
		respondWithJSON(w, http.StatusNotFound, GetResponse{
			Query:    make(map[string]string),
			Data:     make(map[string]string),
			Success:  false,
			Messages: append(messages, "Folder not found"),
		})
		return
	}

	// Is this an insert or an update?
	val, _ := json.Marshal(data)
	_hash := md5.Sum(val)
	hash := _hash[:] // convert from [16]byte to []byte

	var existingKey Now
	var existingData Now
	var countKey int
	var countHash int
	justKeyResult := (s.DB.
		Where("key = ?", key).
		Find(&existingKey).
		Count(&countKey))

	_ = (justKeyResult.
		Where("hash = ?", hash).
		Find(&existingData).
		Count(&countHash))

	var event EventType
	if countKey == 0 {
		event = Insert
	} else if countHash == 0 {
		event = Update
	} else {
		event = NoChange
	}

	messages = append(messages, fmt.Sprint(existingKey))
	messages = append(messages, fmt.Sprint(existingData))
	messages = append(messages, fmt.Sprint("event is ", eventTypeToString(event)))

	marshalledD, _ := json.Marshal(data)

	var success bool
	var msg string

	switch event {
	case Insert:
		success, msg = s.postInsertEvent(matchedFolder, key, hash, marshalledD, event)
		break
	case Update:
		success, msg = s.postUpdateEvent(matchedFolder, key, hash, existingData, marshalledD, event)
		break
	case NoChange:
		success, msg = s.postNoChangeEvent(matchedFolder, key, hash, existingData, marshalledD, event)
		break
	}

	if !success {
		messages = append(messages, fmt.Sprint(msg))
		messages = append(messages, "Failed")

		respondWithJSON(w, http.StatusInternalServerError, GetResponse{
			Query:    vars,
			Data:     data,
			Success:  false,
			Messages: messages,
		})
		return
	}

	respondWithJSON(w, http.StatusCreated, GetResponse{
		Query:    vars,
		Data:     data,
		Success:  true,
		Messages: append(messages, "Created Successfully"),
	})
	return
}

func (s *Samwise) postInsertEvent(
	matchedFolder Folder,
	key string,
	hash []byte,
	marshalledD []byte,
	event EventType) (bool, string) {

	// Start transaction
	tx := s.DB.Begin()

	// add record
	record := Record{
		Folder:      matchedFolder,
		Key:         key,
		Data:        marshalledD,
		LastTouched: time.Now(),
	}

	result := tx.Create(&record)
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create record %s", result.Error)
	}
	// add event to event table
	result = tx.Create(&Event{
		Record:    record,
		EventType: event,
	})
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create Event %s", result.Error)
	}
	// add now
	result = tx.Create(&Now{
		Record: record,
		Folder: matchedFolder,
		Key:    key,
		Hash:   hash,
	})
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create Now %s", result.Error)
	}
	tx.Commit()
	return true, "Insert Successfull"
}

func (s *Samwise) postUpdateEvent(
	matchedFolder Folder,
	key string,
	hash []byte,
	existingData Now,
	marshalledD []byte,
	event EventType) (bool, string) {

	// Start transaction
	tx := s.DB.Begin()

	// add record
	record := Record{
		Folder:      matchedFolder,
		Key:         key,
		Data:        marshalledD,
		LastTouched: time.Now(),
	}
	result := tx.Create(&record)
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create Record %s", result.Error)
	}
	// add event to event table
	result = tx.Create(&Event{
		Record:    record,
		EventType: event,
	})
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create Event %s", result.Error)
	}

	// change now to represent new data
	existingData.Record = record
	existingData.Hash = hash
	result = tx.Save(&existingData)

	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to update Now %s", result.Error)
	}
	tx.Commit()
	return true, ""
}

func (s *Samwise) postNoChangeEvent(
	matchedFolder Folder,
	key string,
	hash []byte,
	existingData Now,
	marshalledD []byte,
	event EventType) (bool, string) {

	// Start transaction
	tx := s.DB.Begin()

	// in the case of no change

	// update record touched at time

	existingData.Record.LastTouched = time.Now()
	result := tx.Save(&existingData)
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to update record %s", result.Error)
	}

	// add event to event table
	result = tx.Create(&Event{
		Record:    existingData.Record,
		EventType: event,
	})
	if result.Error != nil {
		tx.Rollback()
		return false, fmt.Sprintf("Failed to create event %s", result.Error)
	}

	tx.Commit()
	return true, ""
}

// RequestLogger ...
func RequestLogger(targetMux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		targetMux.ServeHTTP(w, r)

		// log request by who(IP address)
		requesterIP := r.RemoteAddr

		log.Printf(
			"%s\t\t%s\t\t%s\t\t%v",
			r.Method,
			r.RequestURI,
			requesterIP,
			time.Since(start),
		)
	})
}

func main() {
	s := Samwise{}
	s.Initialize()
	// s.DB.LogMode(true)
	s.Run(":8080")
}
