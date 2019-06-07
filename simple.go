package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	Folder Folder
	Key    string
	Data   []byte
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

func createFolder(dbHandle *gorm.DB, folderName string) *Folder {
	// TODO: check if already exists

	dbHandle.Create(&Folder{
		Name: folderName,
	})

	// Check if it created successfully
	// Move this to unit test
	var foundFolder Folder
	dbHandle.First(&foundFolder, 0)
	foundFolderJ, _ := json.Marshal(foundFolder)
	fmt.Println("Printing the folder...")
	fmt.Println(string(foundFolderJ))
	return &foundFolder
}

func createRecord(dbHandle *gorm.DB, record *Record) *Record {
	dbHandle.Create(record)

	var foundRecord Record
	dbHandle.First(&foundRecord, 1)
	recJ, _ := json.Marshal(foundRecord)
	fmt.Println(string(recJ))
	return &foundRecord
}

func main() {

	db := getDBHandler()
	defer db.Close()
	// Create
	ptrfolder := createFolder(db, "test")

	ptrrecord := createRecord(db, &Record{
		Folder: *ptrfolder,
		Key:    "example",
		Data:   []byte(`{"Name":"Bob","Food":"Pickle"}`),
	})

	// TODO check searching
	// db.First(&product, "code = ?", "L1212") // find product with code l1212

	// Find the folder metadata associated with this
	var assocfolder Folder
	db.Model(ptrrecord).Related(&assocfolder)
	folderJ, _ := json.Marshal(assocfolder)
	fmt.Println(string(folderJ))

	// Update - update product's price to 2000
	// db.Model(&product).Update("Price", 2000)

	// Delete - delete product
	db.Delete(ptrrecord)
	db.Delete(ptrfolder)
}
